package client
import (
	"net"
	"fmt"
	"bufio"
	"time"
	"github.com/tigeress/goredis/command"
	"github.com/tigeress/goredis/protos"
	"strconv"
	"strings"
	"os"
	"io"
	"github.com/golang/protobuf/proto"
)
var workerNum=1
var conn net.Conn
var writeBuffer map[string]string=make(map[string]string)
var addBuffer map[string]float64=make(map[string]float64)
func TestClientProto(){
	time.Sleep(1000 * time.Millisecond)
	conn=GetConnection("localhost:7777")
	//Set("a","1")
	//fmt.Println(Get("a"))
	//Add("a","1")
	//fmt.Println(Get("a"))
	keys:=Iterate()
	fmt.Println(keys)
	for _,key:= range keys{
		fmt.Println(key+":  "+Get(key+".rank"))
	}
	time.Sleep(1000*time.Millisecond)
}

func FlushBuffer(oper string){
	if oper=="Set" {
		for k,v:=range writeBuffer{
			Set(k,v)
		}
	}
	if oper=="Add" {
		for k,v:=range addBuffer{
			Add(k,strconv.FormatFloat(v,'f',-1,32))
		}
	}
	writeBuffer=make(map[string]string)
	addBuffer=make(map[string]float64)
}
func PageRank(){
	conn=GetConnection("localhost:7777")
	//cache keys and nodes
	nodesCache :=make(map[string][]string)
	countCache:=make(map[string]int)
	keys:=Iterate()
	fmt.Println("keys length:")
	fmt.Println(len(keys))
	for i:=0;i<2;i++{
		//计算每个节点的rank
		for _,key:= range keys{
			adjs:= strings.Split(Get(key+".nodes")," ")
			nodesCache[key]=adjs
			countCache[key],_= strconv.Atoi(Get(key+".count"))
		}
	}
	fmt.Println("cache ok")
	for i:=0;i<2;i++{
		for {
			time.Sleep(100*time.Millisecond)
			if Get("Done")=="1"{
				return
			}
			ifcontinue:=Get("Flush")
			if ifcontinue=="0"||ifcontinue==""{
				break
			}
		}
		//计算每个节点的rank
		for _,key:= range keys{
			if countCache[key]!=0{
				rank,_:=strconv.ParseFloat(Get(key+".rank"),32)
				for _,adj := range nodesCache[key]{
					gradient:=rank/float64(countCache[key])
					AsyncAdd(adj+".gradient",gradient)
				}
			}
		}
		FlushBuffer("Add")
		fmt.Println("one turn compute complete")
		//查看是否所有节点都已经计算完毕。然后每个节点都把crank=nrank。
		//每个节点都同步完成后，再进行下一个超级步
		Flush()
		time.Sleep(1000*time.Millisecond)
	}

}

func LoadWebGraph(){
	time.Sleep(1000*time.Millisecond)
	conn=GetConnection("localhost:7777")
	file,_:=os.Open("/Users/clive/Downloads/web-Stanford.txt")
	reader:=bufio.NewReader(file)
	totalCount:=0
	cache:=make(map[string]string)
	var keys []string
	for {
		line,_,err:=reader.ReadLine()
		if err==io.EOF {
			break
		}
		//fmt.Println("line: "+string(line))
		if line[0]!='#' {
			a:=strings.Split(string(line),"\t")
			prevCountStr:=cache[a[0]+".count"]
			var prevCount int=0
			if prevCountStr!=""&&prevCountStr!="0"{
				prevCount,_=strconv.Atoi(prevCountStr)
				cache[a[0]+".nodes"]=cache[a[0]+".nodes"]+" "+a[1]
			}else if prevCountStr!="0"{
				keys=append(keys,a[0])
				totalCount++
				//Set(a[0]+".nodes",a[1])
				cache[a[0]+".nodes"]=a[1]
			}else{
				cache[a[0]+".nodes"]=a[1]
			}
			//对a[1]也要进行操作
			prevCountStr1:=cache[a[1]+".count"]
			var prevCount1 int=0
			if prevCountStr1!=""{
				prevCount1,_=strconv.Atoi(prevCountStr1)
			}else{
				keys=append(keys,a[1])
				totalCount++
			}
			cache[a[1]+".count"]=strconv.Itoa(prevCount1)
			//Set(a[0]+".count",strconv.Itoa(prevCount+1))
			cache[a[0]+".count"]=strconv.Itoa(prevCount+1)
		}
	}
	//Set("totalCount", strconv.Itoa(totalCount))
	fmt.Println("total:"+strconv.Itoa(totalCount))
	writeBuffer=cache
	for _,k:= range keys{
		writeBuffer[k+".rank"]=strconv.FormatFloat(1/float64(totalCount),'f',-1,32)
	} 
	FlushBuffer("Set")
	//conn.Close()
	time.Sleep(1000*time.Millisecond)
}

func GetConnection(server string) net.Conn{
	conn,err :=net.Dial("tcp",server)
	if err!=nil {
		fmt.Println(err)
	}
	return conn
}
func Flush() string{
	iType:=protos.Type(protos.Type_value["Flush"])
	command:=&protos.Command {
		Type:&iType, 
	}
	commandBytes,_:=proto.Marshal(command)
	conn.Write(commandBytes)
	//Create a data buffer of type byte slice with capacity of 4096
	data := make([]byte, 4096)
	n,_:=conn.Read(data)
	response:= new(protos.Response)
	proto.Unmarshal(data[0:n], response)
	//fmt.Println("receive response:  "+response.String())
	return response.String()
}
func Get(key string) string{
	iType:=protos.Type(protos.Type_value["Get"])
	command:=&protos.Command {
		Type:&iType, 
		Key: &key,
	}
	commandBytes,_:=proto.Marshal(command)
	conn.Write(commandBytes)
	//Create a data buffer of type byte slice with capacity of 4096
	data := make([]byte, 4096)
	    //Read the data waiting on the connection and put it in the data buffer
	n,_:= conn.Read(data)
//	fmt.Println("Decoding Protobuf message")
	response:= new(protos.Response)
	proto.Unmarshal(data[0:n], response)
	//fmt.Println("receive response:  "+response.String())
	if response.Value==nil{
		return ""
	}else{
		return response.Value[0]
	}
}

func Set(key string,value string) string{
	iType:=protos.Type(protos.Type_value["Set"])
	command:=&protos.Command {
		Type:&iType, 
		Key: &key,
		Value: &value,
	}
	commandBytes,_:=proto.Marshal(command)
	conn.Write(commandBytes)
	//Create a data buffer of type byte slice with capacity of 4096
	data := make([]byte, 4096)
	    //Read the data waiting on the connection and put it in the data buffer
	n,_:=conn.Read(data)
	response:= new(protos.Response)
	proto.Unmarshal(data[0:n], response)
	//fmt.Println("receive response:  "+response.String())
	return response.String()
}
func SendCommand(conn net.Conn,cmd command.Command) string{
	return ""
}
func AsyncAdd(key string, value float64) {
	if addBuffer[key]!=0{
		sum:=value+addBuffer[key]
		addBuffer[key]=sum
	}else{
		addBuffer[key]=value
	}
}
func Add(key string,value string) string{
	iType:=protos.Type(protos.Type_value["Add"])
	command:=&protos.Command {
		Type:&iType, 
		Key: &key,
		Value: &value,
	}
	commandBytes,_:=proto.Marshal(command)
	conn.Write(commandBytes)
	//Create a data buffer of type byte slice with capacity of 4096
	data := make([]byte, 4096)
	    //Read the data waiting on the connection and put it in the data buffer
	n,_:=conn.Read(data)
	response:= new(protos.Response)
	proto.Unmarshal(data[0:n], response)
	//fmt.Println("receive response:  "+response.String())
	return response.String()
}
func Iterate() []string{
	iType:=protos.Type(protos.Type_value["Iterate"])
	command:=&protos.Command {
		Type:&iType, 
	}
	commandBytes,_:=proto.Marshal(command)
	conn.Write(commandBytes)
	//Create a data buffer of type byte slice with capacity of 4096
	data := make([]byte, 409600)
	n,_:= conn.Read(data)
	println("iterate read bytes length:"+strconv.Itoa(n))
	response:= new(protos.Response)
	proto.Unmarshal(data[0:n], response)
	//fmt.Println("receive response:  "+response.String())
	return response.Value
}

