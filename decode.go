/*
	Package bencode implements reading and writing of 'bencoded'
	object streams used by the Bittorent protocol.
     Originally from https://github.com/sheepa/videobit/tree/master/bencode
*/
package bencode

import (
	"fmt"
	"strconv"
)

//type List []interface{}
//type Dict map[string]interface{}

//A Decoder reads and decodes bencoded objects from an input stream.
//It returns objects that are either an "Integer", "String", "List" or "Dict".
//
//Example usage:
//	d := bencode.NewDecoder([]byte("i23e4:testi123e"))
//	for !p.Consumed {
//		o, _ := p.Decode()
//		fmt.Printf("obj(%s): %#v\n", reflect.TypeOf(o).Name, o)
//	}
type Decoder struct {
	stream   []byte
	pos      int
	Consumed bool //true if we have consumed all tokens
	Last     int  //overall size to catch buffer underruns
}

//NewDecoder creates a new decoder for the given token stream
func NewDecoder(b []byte) *Decoder {
	return &Decoder{b, 0, false, 0}
}

//Decode reads one object from the input stream
func (self *Decoder) Decode() (res interface{}, err error) {
	res, err = self.nextObject()
	if err == nil {
		self.Last = self.pos
		//fmt.Printf("NOPE %v -- %+v\n", self.pos, err)
	}
	return
}

//DecodeAll reads all objects from the input stream
func (self *Decoder) DecodeAll() (res []interface{}, err error) {
	for {
		o, e := self.nextObject()
		if e != nil {
			err = e
			return
		}
		res = append(res, o)
		if self.pos >= len(self.stream) {
			break
		}
	}
	return
}

func (self *Decoder) GetPos() int {
	return self.pos
}

//fetch the next object at position 'pos' in 'stream'
func (self *Decoder) nextObject() (res interface{}, err error) {
	if self.Consumed {
		return nil, fmt.Errorf("This parser's token stream is consumed!")
	}
	//println(self.pos)
	switch c := self.stream[self.pos]; {
	case c == 'i':
		res, err = self.nextInteger()
	case c >= '0' && c <= '9':
		res, err = self.nextString()
	case c == 'l':
		res, err = self.nextList()
	case c == 'd':
		res, err = self.nextDict()
	default:
		res = nil
		err = fmt.Errorf("Couldn't parse '" + string(self.stream) + "' ... '" + string(self.stream[self.pos]) + "'")
	}
	if self.pos >= len(self.stream) {
		self.Consumed = true
	}
	//fmt.Printf("%+v\n",err)
	//println(self.pos)

	//self.Last = self.pos
	return
}

//fetches next integer from stream and advances pos pointer
func (self *Decoder) nextInteger() (res int, err error) {
	if self.pos < len(self.stream) {
		if self.stream[self.pos] != 'i' {
			return 0, fmt.Errorf("No starting 'i' found")
		}
	} else {
		err = fmt.Errorf("Reached the end of the stream")
		return
	}
	idx := self.pos + 1
	for {
		if self.stream[idx] == 'e' {
			break
		}

		//check for bytes != '-' and '0'..'9'
		if (self.stream[idx] < '0' || self.stream[idx] > '9') && self.stream[idx] != '-' {
			err = fmt.Errorf("Invalid byte '" + string(self.stream[idx]) + "' in encoded integer.")
			return
		}

		idx++
		if idx >= len(self.stream) {
			return 0, fmt.Errorf("No ending 'e' found")
		}
	}

	s := string(self.stream[self.pos+1 : idx])
	r, err := strconv.Atoi(s)
	if err != nil {
		return
	}
	res = r
	self.pos = idx + 1

	return
}

//fetches next string from stream and advances pos pointer
func (self *Decoder) nextString() (res string, err error) {

	if self.pos < len(self.stream) {
		if self.stream[self.pos] < '0' || self.stream[self.pos] > '9' {
			err = fmt.Errorf("No string length determinator found")
			return
		}
	} else {
		err = fmt.Errorf("Reached the end of the stream")
		return
	}

	//fmt.Printf("pos: %v -- len: %v\n",self.pos,len(self.stream))
	len_start := self.pos
	len_end := self.pos

	//scan length
	for {
		if self.stream[len_end] == ':' {
			break
		}
		len_end++
		/*println(len_start)
		println(len_end)
		println(self.pos)
		println(self.size)*/

		if len_end >= len(self.stream) {
			err = fmt.Errorf("No string found ...")
			return
		}
	}

	l, e := strconv.Atoi(string(self.stream[len_start:len_end]))
	if e != nil {
		err = fmt.Errorf("Couldn't parse string length specifier: " + e.Error())
		return
	}
	if l >= len(self.stream[len_end:]) {
		err = fmt.Errorf("Specified length longer than data buffer ...")
		return
	}

	len_end++ //skip the ':'
	res = string(self.stream[len_end : len_end+l])
	err = nil
	self.pos = len_end + l
	return
}

//fetches a list (and its contents) from stream and advances pos
func (self *Decoder) nextList() (res []interface{}, err error) {
	if self.pos < len(self.stream) {
		if self.stream[self.pos] != 'l' {
			err = fmt.Errorf("This is not a list!")
			return
		}
	} else {
		err = fmt.Errorf("Reached the end of the stream")
		return
	}
	self.pos++ //skip 'l'
	for {
		o, e := self.nextObject()
		if e != nil {
			err = e
			return
		}
		res = append(res, o)
		if self.stream[self.pos] == 'e' {
			self.pos++ //skip 'e'
			break
		}
	}
	return
}

//fetches a dict
//bencoded dicts must have their keys sorted lexically. but I guess
//we can ignore that and work with unsorted maps. (wtf?! sorted maps ...)
func (self *Decoder) nextDict() (res map[string]interface{}, err error) {
	if self.pos < len(self.stream) {
		if self.stream[self.pos] != 'd' {
			err = fmt.Errorf("This is not a dict!")
			return
		}
	} else {
		err = fmt.Errorf("Reached the end of the stream")
		return
	}
	res = make(map[string]interface{})

	self.pos++ //skip 'd'
	for {
		//fmt.Printf("pos: %v -- len: %v\n",self.pos,len(self.stream))
		key, e := self.nextString()
		if e != nil {
			err = e
			return
		}
		val, e := self.nextObject()
		if e != nil {
			err = e
			return
		}
		//fmt.Printf("key: %s\nval: %#v\n", key, val)
		res[string(key)] = val
		if self.stream[self.pos] == 'e' {
			self.pos++ //skip 'e'
			break
		}
	}
	return
}
