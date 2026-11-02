// Prepare metadata using verified file digests, category grouping,
// lexical traversal within each category, sorted category list and integer size.
package main
import (
 "bufio"
 "data-ownership-system/dataset"
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
 "sort"
)
func check(e error) { if e!=nil { panic(e) } }
func main() {
 src,dst:=os.Args[1],os.Args[2];check(os.MkdirAll(dst,0755))
 for _,name:=range []string{"coco","clevr","librispeech"} {
  f,e:=os.Open(filepath.Join(src,name+".jsonl"));check(e)
  groups:=map[string][]dataset.DataItemFile{};total:=int64(0);count:=0
  s:=bufio.NewScanner(f)
  for s.Scan() {
   var v struct{URL,Digest,Category string;Size int64};check(json.Unmarshal(s.Bytes(),&v))
   groups[v.Category]=append(groups[v.Category],dataset.DataItemFile{URL:v.URL,Hash:v.Digest})
   total+=v.Size;count++
  };check(s.Err());check(f.Close())
  names:=[]string{};items:=[]dataset.DataItemFile{}
  for cat,rows:=range groups { // Category-group iteration order is unspecified.
   names=append(names,cat)
   sort.Slice(rows,func(i,j int)bool{return rows[i].URL<rows[j].URL})
   items=append(items,rows...)
  }
  sort.Strings(names)
  host:=map[string]string{"coco":"images.cocodataset.org","clevr":"cs.stanford.edu","librispeech":"openslr.org"}[name]
  data:=dataset.DatasetFile{Config:&dataset.DatasetConfig{NumItems:count,NumTypes:len(names),ItemSize:int(total/int64(count)),CompanyName:host,Protocol:"http"},DataTypes:names,Items:items}
  path:=filepath.Join(dst,name+".json")
  if _,e:=os.Stat(path);e==nil { panic("refuse to replace already frozen input: "+path) }
  out,e:=os.Create(path);check(e);check(json.NewEncoder(out).Encode(data));check(out.Close())
  fmt.Printf("%s: %d items, %d categories, integer mean bytes %d\n",name,count,len(names),data.Config.ItemSize)
 }
}
