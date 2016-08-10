package csv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

// Unmarshaller allows you to customize the unmarshal process of a field in CSV
// file
type Unmarshaller interface {
	UnmarshalCSV(string) error
}

var (
	intKindToSize = map[reflect.Kind]int{
		reflect.Int:   0,
		reflect.Int8:  8,
		reflect.Int16: 16,
		reflect.Int32: 32,
		reflect.Int64: 64,
	}
	uintKindToSize = map[reflect.Kind]int{
		reflect.Uint:   0,
		reflect.Uint8:  8,
		reflect.Uint16: 16,
		reflect.Uint32: 32,
		reflect.Uint64: 64,
	}
	floatKindToSize = map[reflect.Kind]int{
		reflect.Float32: 32,
		reflect.Float64: 64,
	}
        created = map[string]int{}
)

// Decoder is a wrap around csv.Reader
type Decoder struct {
	*csv.Reader
}

// NewDecoder will create a new Decoder to be used
func NewDecoder(r io.Reader) *Decoder {
	dec := &Decoder{csv.NewReader(r)}
	dec.TrimLeadingSpace = true
	return dec
}

// Decode will decode the next line in CSV file to v
func (dec *Decoder) Decode(v interface{}) error {
	var (
		err    error
		record []string
		fn     float64
		in     int64
		un     uint64
	)
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
		return errors.New("Decode() expect a pointer to a struct as parameter")
	}

	// the struct
	s := rv.Elem()

	record, err = dec.Read()
	if err != nil {
		return err
	}

	if s.NumField() != len(record) {
		return fmt.Errorf("mismatch length of record: expect %d, get %d", s.NumField(), len(record))
	}

	for i, fValue := range record {
		f := s.Field(i)
		fName := s.Type().Field(i).Name
		if !f.CanSet() {
			return fmt.Errorf("field %q is not settable", fName)
		} else if !f.IsValid() {
			return fmt.Errorf("field %q is not valid", fName)
		}
		// Make sure pointers are properly initialized to nil value
		if f.Kind() == reflect.Ptr && f.IsNil() {
			f.Set(reflect.New(f.Type().Elem()))
		}
		// Only test Unmarshaller interface when it's a pointer and have at
		// least one method.
		if f.Kind() == reflect.Ptr && f.NumMethod() != 0 {
			if x, ok := f.Interface().(Unmarshaller); ok {
				if err = x.UnmarshalCSV(fValue); err != nil {
					return err
				}
				continue
			}
		}
		k := f.Type().Kind()
		if size, ok := intKindToSize[k]; ok {
			in = 0
			if fValue != "" {
				if in, err = strconv.ParseInt(fValue, 10, size); err != nil {
					return fmt.Errorf("failed in parsing %q: %v", fName, err)
				}
			}
			f.SetInt(in)
			continue
		} else if size, ok := uintKindToSize[k]; ok {
			un = 0
			if fValue != "" {
				if un, err = strconv.ParseUint(fValue, 10, size); err != nil {
					return fmt.Errorf("failed in parsing %q: %v", fName, err)
				}
			}
			f.SetUint(un)
			continue
		} else if size, ok := floatKindToSize[k]; ok {
			fn = 0.0
			if fValue != "" {
				if fn, err = strconv.ParseFloat(fValue, size); err != nil {
					return fmt.Errorf("failed in parsing %q: %v", fName, err)
				}
			}
			f.SetFloat(fn)
			continue
		} else if k == reflect.String {
			f.SetString(fValue)
			continue
		}
		return fmt.Errorf("don't know how to decode field %q", fName)
	}
	return nil
}

func (dec *Decoder) DeepUnmarshalCSV(v interface{}) error {
	var (
		err    error
		record []string
	)
        //tv := reflect.TypeOf(v)
	sv := reflect.ValueOf(v)
	if sv.Kind() != reflect.Ptr || sv.IsNil() || sv.Elem().Kind() != reflect.Struct {
		return errors.New("Decode() expect a pointer to a struct as parameter")
	}

	record, err = dec.Read()
	if err != nil {
		return err
	}

        count, err := DeepCount(v)
        if err != nil {
                return fmt.Errorf("Count error: %q: ", err)
        }
	if count != len(record) {
		return fmt.Errorf("Mismatch length of record: expect %d, get %d", count, len(record))
	}

	return DeepUnmarshal(v, record)

}

func DeepUnmarshal(v interface{}, record []string) error {
        var (
                err error
                i,j int
                tail string
        )
        tv := reflect.TypeOf(v)
	sv := reflect.ValueOf(v)
	if sv.Kind() != reflect.Ptr || sv.IsNil() || sv.Elem().Kind() != reflect.Struct {
		return errors.New("Decode() expect a pointer to a struct as parameter")
	}
	// the struct
        t := tv.Elem()
	s := sv.Elem()

        for i,j = 0,0; i < t.NumField(); i,j = i+1,j+1 {
                f := s.Field(i)
                tf := t.Field(i)
                ft := t.Field(i).Type
		fName := tf.Name //s.Type().Field(i).Name
		if !f.CanSet() {
			return fmt.Errorf("field %q is not settable", fName)
		} else if !f.IsValid() {
			return fmt.Errorf("field %q is not valid", fName)
		}
                if ft.Kind() == reflect.Struct {
                        c := reflect.New(ft)
                        f.Set(c.Elem())
                        if err = DeepUnmarshal(f.Addr().Interface(), record[j:]); err != nil {
                                return fmt.Errorf("Unmarshal error: %q ", err)
                        } else {
                                if cnt, err := DeepCount(f.Addr().Interface()); err != nil{
                                        return fmt.Errorf("Counting error: %q", err)
                                } else {
                                        j += int(cnt)
                                        tail += fName
                                }
                                continue
                        }
                } else if ft.Kind() == reflect.Slice {
                        if add, err := strconv.ParseInt(tf.Tag.Get("cap"),10,0); err != nil{
                                return fmt.Errorf("Counting error: %q Count so far: %d Tag: %q", err, j, tf.Tag.Get("cap"))
                        } else {
                                f.Set(reflect.MakeSlice(ft, int(add), int(add)))
                                for n:=0; n < int(add); n++ {
                                        if err = SetFieldWithValue(ft, f.Slice(n,n), record[j+n]); err != nil {
                                                return fmt.Errorf("Error setting slice: %q, %q, %q, %q, %d, %d", f.Slice(n,n), record, err, fName, j,n)
                                        }
                                        tail += fName
                                }
                                j += int(add) - 1
                        }
                } else {
                        if err := SetFieldWithValue(ft, f, record[j]); err != nil {
                                return fmt.Errorf("Field set error: %q", err)
                        } else {
                                tail += fName
                                continue
                        }
                }
	}
        return nil //fmt.Errorf("Last i,j : %d,%d",i, j)
}

func DeepCount(v interface{}) (int, error) {
        count := 0
        typ := reflect.TypeOf(v).Elem()
        if typ.Kind() == reflect.Struct {
                for i := 0; i < typ.NumField(); i++ {
                        f := typ.Field(i) //StructField
                        ft := f.Type
                        if ft.Kind() == reflect.Struct {
                                if c, err := DeepCount(reflect.New(ft).Interface()); err != nil{  //recursive call   
                                        return 0, fmt.Errorf("Counting error: %q", err)
                                } else {
                                        count += int(c)
                                }
                        } else if ft.Kind() == reflect.Slice {
                                if add, err := strconv.ParseInt(f.Tag.Get("cap"),10,0); err != nil{
                                        return 0, fmt.Errorf("Counting error: %q Count so far: %q Tag: %q", err, count, f.Tag.Get("cap"))
                                } else {
                                        count += int(add)
                                }
                        } else {
                                count++
                        }
                }
        }
        return count, nil //fmt.Errorf("Returning: %q Count: %q", typ.Kind(), typ.NumField())
}

func SetFieldWithValue(ft reflect.Type,  f reflect.Value, fValue string) error {
        var (
                err error
                jn int
                st string
                fn     float64
                in     int64
                un     uint64
        )
        // set field value
        //fName := ft.Name()
        fName := f.Type().Name()
        k := ft.Kind()
        if size, ok := intKindToSize[k]; ok {
                in = 0
                if fValue != "" {
                        if in, err = strconv.ParseInt(fValue, 10, size); err != nil {
                                return fmt.Errorf("failed in parsing %q: %v", fName, err)
                        }
                }
                f.SetInt(in)
                return nil
        } else if size, ok := uintKindToSize[k]; ok {
                un = 0
                if fValue != "" {
                        if un, err = strconv.ParseUint(fValue, 10, size); err != nil {
                                return fmt.Errorf("failed in parsing %q: %v", fName, err)
                        }
                }
                f.SetUint(un)
                return nil
        } else if size, ok = floatKindToSize[k]; ok {
                fn = 0.0
                if fValue != "" {
                        if fn, err = strconv.ParseFloat(fValue, size); err != nil {
                                return fmt.Errorf("failed in parsing %q: %v", fName, err)
                        }
                }
                f.SetFloat(fn)
                return nil
        } else if k == reflect.String {
                f.SetString(fValue)
                return nil
        } else if k == reflect.Slice {
                switch ft {
                        case reflect.SliceOf(reflect.TypeOf(jn)):
                                if in, err = strconv.ParseInt(fValue, 10, 0); err != nil {
                                        return fmt.Errorf("failed in parsing %q: %v %q", fName, err, ft)
                                }
                                reflect.Append(f, reflect.ValueOf(int(in)))
                        case reflect.SliceOf(reflect.TypeOf(st)):
                                reflect.Append(f, reflect.ValueOf(fValue))
                        default:
                                reflect.Append(f, reflect.ValueOf(fValue))
                }
                return nil
        }
        return fmt.Errorf("don't know how to decode field %q %q", fName, k)
        // end set field value
}

/*func csvMarshal(csv []string) *Good {
        id, err := strconv.ParseInt(csv[0], 10, 64)
        handle(err)
        prc, err:= strconv.ParseFloat(csv[6],64)
        handle(err)
        ftd, err := strconv.ParseBool(csv[9])
        handle(err)
        hdn, err :=  strconv.ParseBool(csv[10])
        handle(err)
        tax, err := strconv.ParseFloat(csv[12],64)
        handle(err)
        price, err := strconv.ParseFloat(csv[13],64)
        handle(err)
        stock, err := strconv.Atoi(csv[14])
        handle(err)
        rel1, err := strconv.ParseInt(csv[15], 10, 64)
        handle(err)
        rel2, err := strconv.ParseInt(csv[16], 10, 64)
        handle(err)
        rel3, err := strconv.ParseInt(csv[17], 10, 64)
        handle(err)
        rel4, err := strconv.ParseInt(csv[18], 10, 64)
        handle(err)
        rel5, err := strconv.ParseInt(csv[19], 10, 64)
        handle(err)
        rel6, err := strconv.ParseInt(csv[20], 10, 64)
        handle(err)
        prcs1, err := strconv.ParseFloat(csv[21],64)
        handle(err)
        prcs2, err := strconv.ParseFloat(csv[22],64)
        handle(err)
        prcs3, err := strconv.ParseFloat(csv[23],64)
        handle(err)
        prcs4, err := strconv.ParseFloat(csv[24],64)
        handle(err)
        prcs5, err := strconv.ParseFloat(csv[25],64)
        handle(err)
        prcs6, err := strconv.ParseFloat(csv[26],64)
        handle(err)
        vols1, err := strconv.Atoi(csv[27])
        vols2, err := strconv.Atoi(csv[28])
        vols3, err := strconv.Atoi(csv[29])
        vols4, err := strconv.Atoi(csv[30])
        vols5, err := strconv.Atoi(csv[31])
        vols6, err := strconv.Atoi(csv[32])
        pd1 := GoodDeets{ DescDetails:csv[11], Tax:tax, Price:price, Stock:stock, Related:[]int64{rel1,rel2,rel3,rel4,rel5,rel6}, Prices: []float64{prcs1,prcs2,prcs3,prcs4,prcs5,prcs6}, Volumes: []int{vols1,vols2,vols3,vols4,vols5,vols6}, ParameterNames: []string{csv[33],csv[34],csv[35],csv[36],csv[37],csv[38],csv[39],csv[40],csv[41],csv[42],csv[43],csv[44]}, ParameterValues: []string{csv[45],csv[46],csv[47],csv[48],csv[49],csv[50],csv[51],csv[52],csv[53],csv[54],csv[55],csv[56]}, Features: []string{csv[57],csv[58],csv[59],csv[60],csv[61],csv[62],csv[63],csv[64],csv[65],csv[66],csv[67],csv[68]}, Items: []string{csv[69],csv[70],csv[71],csv[72],csv[73],csv[74]}, UrlImgs1:csv[75], UrlImgs2:csv[76], UrlImgs3:csv[77], UrlFile:csv[78]}
        return  &Good{ Id: id, Code: csv[1], Category: csv[2], Subcategory: csv[3], Brand: csv[4], Desc: csv[5], Price:prc, Url: csv[7], Urlimg: csv[8], Featured: ftd, Hidden:hdn, Deets: pd1 }
}*/
