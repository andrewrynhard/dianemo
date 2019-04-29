/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package file

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/talos-systems/talos/internal/pkg/chunker"
	"gopkg.in/fsnotify.v1"
)

// Options is the functional options struct.
type Options struct {
	Size int
}

// Option is the functional option func.
type Option func(*Options)

// Size sets the chunk size of the Chunker.
func Size(s int) Option {
	return func(args *Options) {
		args.Size = s
	}
}

// File is a conecrete type that implements the chunker.Chunker interface.
type File struct {
	source  Source
	options *Options
}

// Source is an interface describing the source of a File.
type Source = *os.File

// NewChunker initializes a Chunker with default values.
func NewChunker(source Source, setters ...Option) chunker.Chunker {
	opts := &Options{
		Size: 1024,
	}

	for _, setter := range setters {
		setter(opts)
	}

	return &File{
		source,
		opts,
	}
}

// Read implements ChunkReader.
func (c *File) Read(ctx context.Context) <-chan []byte {
	// Create a buffered channel of length 1.
	ch := make(chan []byte, 1)

	go func(ch chan []byte) {
		defer close(ch)

		offset, err := c.source.Seek(0, io.SeekStart)
		if err != nil {
			return
		}
		buf := make([]byte, c.options.Size)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := c.source.ReadAt(buf, offset)
				if err != nil {
					if err != io.EOF {
						fmt.Printf("read error: %s\n", err.Error())
						break
					}
				}
				offset += int64(n)
				if n != 0 {
					// Copy the buffer since we will modify it in the next loop.
					b := make([]byte, n)
					copy(b, buf[:n])
					ch <- b

					// Clear the buffer.
					for i := 0; i < n; i++ {
						buf[i] = 0
					}
				} else if err := watch(c.source.Name()); err != nil {
					log.Printf("failed to watch: %v\n", err)
					return
				}
			}
		}
	}(ch)

	return ch
}

func watch(path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	// nolint: errcheck
	defer watcher.Close()

	errChan := make(chan error)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				switch event.Op {
				case fsnotify.Write:
					errChan <- nil
					return
				case fsnotify.Remove:
					errChan <- errors.Errorf("file was removed while watching: %s", path)
					return
				default:
					log.Printf("ignoring fsnotify event: %v\n", event)
				}
			case err := <-watcher.Errors:
				errChan <- err
				return
			}
		}
	}()

	if err := watcher.Add(path); err != nil {
		return err
	}

	e := <-errChan

	return e
}
