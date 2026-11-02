package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/sha3"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Row struct {
	URL      string `json:"url"`
	Digest   string `json:"digest"`
	Category string `json:"category"`
	Size     int64  `json:"size"`
}
type Archive struct {
	File        string `json:"file"`
	Bytes       int64  `json:"bytes"`
	SHA256      string `json:"sha256"`
	MD5         string `json:"md5"`
	CRCVerified bool   `json:"crc_verified"`
}
type Dataset struct {
	Name       string         `json:"name"`
	Files      int            `json:"files"`
	Bytes      int64          `json:"bytes"`
	Categories map[string]int `json:"categories"`
	Archives   []Archive      `json:"archives"`
	Excluded   []string       `json:"excluded"`
	Seconds    float64        `json:"preprocess_seconds"`
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
func locate(s string) string {
	if _, e := os.Stat(filepath.Join("datasets", s)); e == nil {
		return filepath.Join("datasets", s)
	}
	return s
}
func checksums(p string) Archive {
	f, e := os.Open(p)
	check(e)
	defer f.Close()
	s := sha256.New()
	m := md5.New()
	n, e := io.Copy(io.MultiWriter(s, m), f)
	check(e)
	return Archive{p, n, hex.EncodeToString(s.Sum(nil)), hex.EncodeToString(m.Sum(nil)), true}
}
func main() {
	check(os.MkdirAll("datasets", 0755))
	reports := []Dataset{}
	for _, name := range []string{"coco", "clevr", "librispeech"} {
		start := time.Now()
		r := Dataset{Name: name, Categories: map[string]int{}}
		out, e := os.Create("datasets/" + name + ".jsonl")
		check(e)
		enc := json.NewEncoder(out)
		add := func(url, cat string, size int64, rd io.Reader) {
			h := sha3.NewLegacyKeccak256()
			n, e := io.Copy(h, rd)
			check(e)
			if n != size {
				panic("size mismatch")
			}
			check(enc.Encode(Row{url, hex.EncodeToString(h.Sum(nil)), cat, size}))
			r.Files++
			r.Bytes += size
			r.Categories[cat]++
		}
		if name != "librispeech" {
			names := []string{"train2017.zip", "val2017.zip", "test2017.zip", "annotations_trainval2017.zip"}
			if name == "clevr" {
				names = []string{"CLEVR_v1.0.zip"}
			}
			for _, file := range names {
				p := locate(file)
				fmt.Println("Reading", p)
				z, e := zip.OpenReader(p)
				check(e)
				for _, f := range z.File {
					if f.FileInfo().IsDir() {
						continue
					}
					path := f.Name
					host := "images.cocodataset.org"
					if name == "clevr" {
						path = strings.TrimPrefix(path, "CLEVR_v1.0/")
						if strings.HasPrefix(path, "images/") {
							path = strings.TrimPrefix(path, "images/")
						} else if !strings.HasPrefix(path, "scenes/") && !strings.HasPrefix(path, "questions/") {
							r.Excluded = append(r.Excluded, f.Name)
							continue
						}
						host = "cs.stanford.edu"
					}
					parts := strings.SplitN(path, "/", 2)
					rd, e := f.Open()
					check(e)
					add("http://"+host+"/"+path, parts[0], int64(f.UncompressedSize64), rd)
					check(rd.Close())
				}
				check(z.Close())
				r.Archives = append(r.Archives, checksums(p))
			}
		} else {
			p := locate("train-other-500.tar.gz")
			fmt.Println("Reading", p)
			f, e := os.Open(p)
			check(e)
			g, e := gzip.NewReader(f)
			check(e)
			t := tar.NewReader(g)
			for {
				h, e := t.Next()
				if e == io.EOF {
					break
				}
				check(e)
				if !h.FileInfo().Mode().IsRegular() {
					continue
				}
				path := strings.TrimPrefix(h.Name, "LibriSpeech/train-other-500/")
				if path == h.Name {
					r.Excluded = append(r.Excluded, h.Name)
					continue
				}
				parts := strings.SplitN(path, "/", 2)
				if len(parts) != 2 {
					r.Excluded = append(r.Excluded, h.Name)
					continue
				}
				add("http://openslr.org/"+path, parts[0], h.Size, t)
			}
			_, e = io.Copy(io.Discard, g)
			check(e)
			check(g.Close())
			check(f.Close())
			a := checksums(p)
			if a.MD5 != "d1a0fd59409feb2c614ce4d30c387708" {
				panic("official LibriSpeech MD5 mismatch")
			}
			r.Archives = append(r.Archives, a)
		}
		check(out.Close())
		r.Seconds = time.Since(start).Seconds()
		sort.Strings(r.Excluded)
		reports = append(reports, r)
		b, _ := json.MarshalIndent(reports, "", "  ")
		check(os.WriteFile("datasets/validation.json", b, 0644))
		fmt.Printf("%s: %d files, %d categories, %.1fs\n", name, r.Files, len(r.Categories), r.Seconds)
	}
}
