// Copyright ©2013 The bíogo.nn Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mnist provides a simple interface to access the MNIST database of handwritten digits.
// The mnist package does not come bundled with the database, but will attempt to download the
// data if it does not already exist in the package's root directory.
//
// More information on MNIST is provided at http://yann.lecun.com/exdb/mnist/.
package mnist

import (
	"compress/gzip"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
)

const (
	xLAB int32 = 0x00000801
	xIMG int32 = 0x00000803
)

// If Logger is not nil, MNIST data retrieval will be logged.
var Logger *log.Logger = log.New(os.Stderr, "mnist: ", log.LstdFlags)

var (
	mnist = []struct {
		url    string
		local  string
		length int64
		md5    string
	}{
		/*
			TRAINING SET IMAGE FILE (train-images-idx3-ubyte):
			[offset] [type]          [value]          [description]
			0000     32 bit integer  0x00000803(2051) magic number
			0004     32 bit integer  60000            number of images
			0008     32 bit integer  28               number of rows
			0012     32 bit integer  28               number of columns
			0016     unsigned byte   ??               pixel
			0017     unsigned byte   ??               pixel
			........
			xxxx     unsigned byte   ??               pixel

			Pixels are organized row-wise. Pixel values are 0 to 255. 0 means background (white), 255 means foreground (black).
		*/
		{
			url:    "http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz",
			length: 9912422,
			md5:    "f68b3c2dcbeaaa9fbdd348bbdeb94873",
		},

		/*
			TRAINING SET LABEL FILE (train-labels-idx1-ubyte):
			[offset] [type]          [value]          [description]
			0000     32 bit integer  0x00000801(2049) magic number (MSB first)
			0004     32 bit integer  60000            number of items
			0008     unsigned byte   ??               label
			0009     unsigned byte   ??               label
			........
			xxxx     unsigned byte   ??               label

			The labels values are 0 to 9.
		*/
		{
			url:    "http://yann.lecun.com/exdb/mnist/train-labels-idx1-ubyte.gz",
			length: 28881,
			md5:    "d53e105ee54ea40749a09fcbcd1e9432",
		},

		/*
			TEST SET IMAGE FILE (t10k-images-idx3-ubyte):
			[offset] [type]          [value]          [description]
			0000     32 bit integer  0x00000803(2051) magic number
			0004     32 bit integer  10000            number of images
			0008     32 bit integer  28               number of rows
			0012     32 bit integer  28               number of columns
			0016     unsigned byte   ??               pixel
			0017     unsigned byte   ??               pixel
			........
			xxxx     unsigned byte   ??               pixel

			Pixels are organized row-wise. Pixel values are 0 to 255. 0 means background (white), 255 means foreground (black).
		*/
		{
			url:    "http://yann.lecun.com/exdb/mnist/t10k-images-idx3-ubyte.gz",
			length: 1648877,
			md5:    "9fb629c4189551a2d022fa330f9573f3",
		},

		/*
			TEST SET LABEL FILE (t10k-labels-idx1-ubyte):
			[offset] [type]          [value]          [description]
			0000     32 bit integer  0x00000801(2049) magic number (MSB first)
			0004     32 bit integer  10000            number of items
			0008     unsigned byte   ??               label
			0009     unsigned byte   ??               label
			........
			xxxx     unsigned byte   ??               label

			The labels values are 0 to 9.
		*/
		{
			url:    "http://yann.lecun.com/exdb/mnist/t10k-labels-idx1-ubyte.gz",
			length: 4542,
			md5:    "ec29112dd5afa0611ce80d1b7f02629c",
		},
	}
)

func init() {
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		if Logger != nil {
			Logger.Fatal("cannot get file location")
		}
		fmt.Fprintf(os.Stderr, "mnist: cannot get file location")
		os.Exit(1)
	}
	dir := filepath.Dir(path)

	if Logger != nil {
		Logger.Print("Checking for MNIST data...")
	}
	cl := &http.Client{}
	for i := range mnist {
		u, err := url.Parse(mnist[i].url)
		isNil(err)
		fn := filepath.Base(u.Path)
		mnist[i].local = filepath.Join(dir, fn)
		if f, err := os.Open(mnist[i].local); err == nil {
			if fs, err := f.Stat(); err == nil && fs.Size() == mnist[i].length {
				hash := md5.New()
				n, err := io.Copy(hash, f)
				isNil(err)
				s := hash.Sum(nil)
				if n == mnist[i].length && fmt.Sprintf("%x", s) == mnist[i].md5 {
					if Logger != nil {
						Logger.Printf(" %s: OK", fn)
					}
					continue
				}
			}
		}
		if Logger != nil {
			Logger.Printf(" %s: Downloading", fn)
		}
		res, err := cl.Get(mnist[i].url)
		isNil(err)
		f, err := os.Create(mnist[i].local)
		isNil(err)
		n, err := io.Copy(f, res.Body)
		if n != mnist[i].length {
			if Logger != nil {
				Logger.Fatalf("length mismatch %d != %d", n, mnist[i].length)
			}
			fmt.Fprintf(os.Stderr, "mnist: length mismatch %d != %d", n, mnist[i].length)
			os.Exit(1)
		}
		isNil(err)
		res.Body.Close()
		f.Close()
	}

	isNil(Train.read(mnist[0].local, mnist[1].local))
	isNil(Test.read(mnist[2].local, mnist[3].local))
}

func isNil(err error) {
	if err != nil {
		if Logger != nil {
			Logger.Fatal(err)
		}
		panic(fmt.Sprintf("mnist: %v", err))
	}
}

var (
	// Train contains the MNIST training set of 60,000 digits with labels.
	Train Set

	// Test contains the MNIST test set of 10,000 digits with labels.
	Test Set
)

// A Set contains a set of labelled digit images.
type Set struct {
	count      int32
	rows, cols int32
	matrix     []byte // count*rows*cols
	labels     []byte // count
}

// Rows returns the number of pixel rows in the images of the data set.
func (s *Set) Rows() int { return int(s.rows) }

// Cols returns the number of pixel columns in the images of the data set.
func (s *Set) Cols() int { return int(s.cols) }

// Len returns the number of labelled images in the data set.
func (s *Set) Len() int { return int(s.count) }

// Index returns the i'th label and image of the data set.
func (s *Set) Index(i int) (label byte, image []byte) {
	stride := int(s.rows * s.cols)
	return s.labels[i], s.matrix[i*stride : (i+1)*stride]
}

func (s *Set) read(images, labels string) error {
	err := s.readImages(images)
	if err != nil {
		return err
	}
	return s.readLabels(labels)
}

func (s *Set) readImages(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer z.Close()

	var magic int32
	err = binary.Read(z, binary.BigEndian, &magic)
	if err != nil {
		return err
	}
	if magic != xIMG {
		return fmt.Errorf("invalid magic number for images: %x", magic)
	}
	for _, v := range []*int32{&s.count, &s.rows, &s.cols} {
		err = binary.Read(z, binary.BigEndian, v)
		if err != nil {
			return err
		}
	}
	s.matrix = make([]byte, s.count*s.rows*s.cols)
	_, err = io.ReadFull(z, s.matrix)

	return err
}

func (s *Set) readLabels(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer z.Close()

	var magic int32
	err = binary.Read(z, binary.BigEndian, &magic)
	if err != nil {
		return err
	}
	if magic != xLAB {
		return fmt.Errorf("invalid magic number for labels: %x", magic)
	}
	var count int32
	err = binary.Read(z, binary.BigEndian, &count)
	if err != nil {
		return err
	}
	if count != s.count {
		return errors.New("mismatched number of labels and images")
	}
	s.labels = make([]byte, s.count)
	_, err = io.ReadFull(z, s.labels)

	return err
}
