package wav

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/geiqin/duration/audio"
	"github.com/geiqin/duration/audio/riff"
)

// Encoder encodes LPCM data into a wav containter.
type Encoder struct {
	w          io.WriteSeeker
	SampleRate int
	BitDepth   int
	NumChans   int

	// A number indicating the WAVE format category of the file. The content of the
	// <format-specific-fields> portion of the ‘fmt’ chunk, and the interpretation of
	// the waveform data, depend on this value.
	// PCM = 1 (i.e. Linear quantization) Values other than 1 indicate some form of compression.
	WavAudioFormat int

	WrittenBytes    int
	frames          int
	pcmChunkStarted bool
	pcmChunkSizePos int
}

// NewEncoder creates a new encoder to create a new wav file.
// Don't forget to add Frames to the encoder before writing.
func NewEncoder(w io.WriteSeeker, sampleRate, bitDepth, numChans, audioFormat int) *Encoder {
	return &Encoder{
		w:              w,
		SampleRate:     sampleRate,
		BitDepth:       bitDepth,
		NumChans:       numChans,
		WavAudioFormat: audioFormat,
	}
}

// AddLE serializes and adds the passed value using little endian
func (e *Encoder) AddLE(src interface{}) error {
	e.WrittenBytes += binary.Size(src)
	return binary.Write(e.w, binary.LittleEndian, src)
}

// AddBE serializes and adds the passed value using big endian
func (e *Encoder) AddBE(src interface{}) error {
	e.WrittenBytes += binary.Size(src)
	return binary.Write(e.w, binary.BigEndian, src)
}

func (e *Encoder) addBuffer(buf *audio.PCMBuffer) error {
	if buf == nil {
		return fmt.Errorf("can't add a nil buffer")
	}

	frameCount := buf.Size()
	buf.CacheInts()
	for i := 0; i < frameCount; i++ {
		// TODO(mattetti): support float encoded wav files
		for j := 0; j < buf.Format.NumChannels; j++ {
			v := buf.Ints[i*buf.Format.NumChannels+j]
			switch e.BitDepth {
			case 8:
				if err := e.AddLE(uint8(v)); err != nil {
					return err
				}
			case 16:
				if err := e.AddLE(int16(v)); err != nil {
					return err
				}
			case 24:
				if err := e.AddLE(audio.Int32toInt24LEBytes(int32(v))); err != nil {
					return err
				}
			case 32:
				if err := e.AddLE(int32(v)); err != nil {
					return err
				}
			default:
				return fmt.Errorf("can't add frames of bit size %d", e.BitDepth)
			}
		}
		e.frames++
	}

	return nil
}

func (e *Encoder) writeHeader() error {
	if e == nil {
		return fmt.Errorf("can't write a nil encoder")
	}
	if e.w == nil {
		return fmt.Errorf("can't write to a nil writer")
	}

	if e.WrittenBytes > 0 {
		return nil
	}

	// riff ID
	if err := e.AddLE(riff.RiffID); err != nil {
		return err
	}
	// file size uint32, to update later on.
	if err := e.AddLE(uint32(42)); err != nil {
		return err
	}
	// wave headers
	if err := e.AddLE(riff.WavFormatID); err != nil {
		return err
	}
	// form
	if err := e.AddLE(riff.FmtID); err != nil {
		return err
	}
	// chunk size
	if err := e.AddLE(uint32(16)); err != nil {
		return err
	}
	// wave format
	if err := e.AddLE(uint16(e.WavAudioFormat)); err != nil {
		return err
	}
	// num channels
	if err := e.AddLE(uint16(e.NumChans)); err != nil {
		return fmt.Errorf("error encoding the number of channels - %v", err)
	}
	// samplerate
	if err := e.AddLE(uint32(e.SampleRate)); err != nil {
		return fmt.Errorf("error encoding the sample rate - %v", err)
	}
	// avg bytes per sec
	if err := e.AddLE(uint32(e.SampleRate * e.NumChans * e.BitDepth / 8)); err != nil {
		return fmt.Errorf("error encoding the avg bytes per sec - %v", err)
	}
	// block align
	if err := e.AddLE(uint16(2)); err != nil {
		return err
	}
	// bits per sample
	if err := e.AddLE(uint16(e.BitDepth)); err != nil {
		return fmt.Errorf("error encoding bits per sample - %v", err)
	}

	return nil
}

// Write encodes and writes the passed buffer to the underlying writer.
// Don't forger to Close() the encoder or the file won't be valid.
func (e *Encoder) Write(buf *audio.PCMBuffer) error {
	if err := e.writeHeader(); err != nil {
		return err
	}

	if !e.pcmChunkStarted {
		// sound header
		if err := e.AddLE(riff.DataFormatID); err != nil {
			return fmt.Errorf("error encoding sound header %v", err)
		}
		e.pcmChunkStarted = true

		// write a temporary chunksize
		e.pcmChunkSizePos = e.WrittenBytes
		if err := e.AddLE(uint32(42)); err != nil {
			return fmt.Errorf("%v when writing wav data chunk size header", err)
		}
	}

	return e.addBuffer(buf)
}

// Close flushes the content to disk, make sure the headers are up to date
// Note that the underlying writter is NOT being closed.
func (e *Encoder) Close() error {
	if e == nil || e.w == nil {
		return nil
	}

	// go back and write total size in header
	if _, err := e.w.Seek(4, 0); err != nil {
		return err
	}
	if err := e.AddLE(uint32(e.WrittenBytes) - 8); err != nil {
		return fmt.Errorf("%v when writing the total written bytes", err)
	}

	// rewrite the audio chunk length header
	if e.pcmChunkSizePos > 0 {
		if _, err := e.w.Seek(int64(e.pcmChunkSizePos), 0); err != nil {
			return err
		}
		chunksize := uint32((int(e.BitDepth) / 8) * int(e.NumChans) * e.frames)
		if err := e.AddLE(uint32(chunksize)); err != nil {
			return fmt.Errorf("%v when writing wav data chunk size header", err)
		}
	}

	// jump back to the end of the file.
	if _, err := e.w.Seek(0, 2); err != nil {
		return err
	}
	switch e.w.(type) {
	case *os.File:
		return e.w.(*os.File).Sync()
	}
	return nil
}
