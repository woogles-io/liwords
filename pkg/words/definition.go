package words

import (
	"io"
	"os"
	"sync"
)

type defSource struct {
	file     *os.File // opened at startup and never closed
	fileSize int64
	blkSize  int64
}

func loadDefinitionSource(filename string) (*defSource, error) {
	// text file of WORD\tdefinition\n with LC_ALL=C sort
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	fileOk := false
	defer func() {
		if !fileOk {
			f.Close()
		}
	}()

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()
	blkSize := getBlkSize(&fileInfo, 4096)

	fileOk = true
	return &defSource{file: f, fileSize: fileSize, blkSize: blkSize}, err
}

var bufPool = sync.Pool{
	New: func() interface{} {
		var x []byte
		return &x
	},
}

func (ds *defSource) bulkDefine(sortedWords []string) (map[string]string, error) {
	f := ds.file
	fileSize := ds.fileSize
	blkSize := ds.blkSize

	pooledBuf := bufPool.Get().(*[]byte)
	if cap(*pooledBuf) < int(blkSize) {
		newPooledBuf := make([]byte, int(blkSize))
		pooledBuf = &newPooledBuf
	}
	defer bufPool.Put(pooledBuf)
	buf := (*pooledBuf)[:blkSize]
	bufMin := int64(0)
	bufMax := int64(0)
	numBlks := (fileSize + blkSize - 1) / blkSize

	// this "reinvents" mmap with a single block cache; it is not as performant, but it is portable
	readByte := func(ofs int64) (byte, error) {
		if ofs >= bufMin && ofs < bufMax {
			return buf[ofs-bufMin], nil
		}
		if ofs < 0 || ofs >= fileSize {
			return 0, io.EOF
		}
		innerOfs := ofs % blkSize
		outerOfs := ofs - innerOfs
		// unlike Seek, ReadAt does not use the file pointer, so it is ok to have multiple goroutines read the same file
		n, err := f.ReadAt(buf, outerOfs)
		if err != nil && err != io.EOF {
			return 0, err
		}
		bufMin = outerOfs
		bufMax = bufMin + int64(n)
		if innerOfs >= int64(n) {
			return 0, io.EOF
		}
		return buf[innerOfs], nil
	}

	ret := make(map[string]string)
	cur := int64(0)

	for _, word := range sortedWords {
		var def []byte
		var ofs int64
		var b byte
		var err error
		lo := cur/blkSize + 1 // exclude current block
		origLo := lo
		// binary search among remaining blocks
		for hi := numBlks; lo < hi; {
			mid := int64(uint64(lo+hi) >> 1)
			ofs = mid * blkSize
			// skip to newline (because this is after the current block)
			for ; ; ofs++ {
				b, err = readByte(ofs)
				if err == io.EOF {
					goto high
				} else if err != nil {
					return nil, err
				} else if b == '\n' {
					ofs++
					// compare the word starting at ofs vs word
					for _, c := range []byte(word) {
						b, err = readByte(ofs)
						if err == io.EOF {
							goto high
						} else if err != nil {
							return nil, err
						} else if b > c {
							goto high
						} else if b < c || b == '\t' {
							goto low
						}
						ofs++
					}
					b, err = readByte(ofs)
					if err == io.EOF {
						goto after_match
					} else if err != nil {
						return nil, err
					} else if b == '\t' {
						goto found
					}
					goto high
				}
			}
		low:
			// block's first word is not greater (it may be found here)
			lo = mid + 1
			continue
		high:
			// block's first word is greater, exclude
			hi = mid
		}
		// compare current block vs binary search result
		if lo == origLo {
			ofs = cur
			goto start_of_possibly_matching_line
		}
		// skip to newline (because this is after the current block)
		ofs = (lo - 1) * blkSize
	before_match:
		for ; ; ofs++ {
			b, err = readByte(ofs)
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			} else if b == '\n' {
				ofs++
				break
			}
		}
	start_of_possibly_matching_line:
		// now ofs is at start of line, linear search from there
		cur = ofs
		for _, c := range []byte(word) {
			b, err = readByte(ofs)
			if err == io.EOF {
				goto after_match
			} else if err != nil {
				return nil, err
			} else if b > c {
				goto after_match
			} else if b < c || b == '\t' {
				goto before_match
			}
			ofs++
		}
		b, err = readByte(ofs)
		if err == io.EOF {
			goto after_match
		} else if err != nil {
			return nil, err
		} else if b == '\t' {
			goto found
		}
		goto after_match
	found:
		// ofs points at the '\t'
		def = make([]byte, 0)
		for ofs++; ; ofs++ {
			b, err = readByte(ofs)
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			} else if b == '\n' {
				ofs++
				break
			} else {
				def = append(def, b)
			}
		}
		ret[word] = string(def)
		cur = ofs
	after_match:
	}

	return ret, nil
}
