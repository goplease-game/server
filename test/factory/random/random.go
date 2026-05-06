// Package random provides utility functions for generating various types of random data.
package random

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand/v2"
	"strings"

	fake "github.com/brianvoe/gofakeit/v7"
	"github.com/ognev-dev/goplease/app"
)

var (
	letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	digits  = []byte("0123456789")
)

// String generates a random alphanumeric string.
// The length of the string defaults to 10 if no argument is provided.
func String(lengthOpt ...int) string {
	length := 10
	if len(lengthOpt) == 1 {
		length = lengthOpt[0]
	}

	alphanum := letters
	alphanum = append(alphanum, digits...)

	s := make([]byte, length)
	for i := range length {
		s[i] = alphanum[rand.IntN(len(alphanum))] //nolint:gosec
	}

	return string(s)
}

// Title generates a random string formatted in title case.
func Title() string {
	return Element([]string{fake.BookTitle(), fake.MovieName()})
}

// Email generates randomly constructed email address.
func Email() string {
	return strings.ToLower(fmt.Sprintf("%s@%s.%s", Letters(6), Letters(6), Letters(3))) //nolint:mnd
}

// Letters generates a random string consisting only of alphabetic characters (a-z, A-Z).
func Letters(length int) string {
	var sb strings.Builder
	sb.Grow(length)

	for range length {
		sb.WriteString(Letter())
	}

	return sb.String()
}

// Digits generates a random string consisting only of numeric characters (0-9).
func Digits(length int) string {
	var sb strings.Builder
	sb.Grow(length)

	for range length {
		sb.WriteString(Digit())
	}

	return sb.String()
}

// Digit returns a single random numeric character (0-9) as a string.
func Digit() string {
	d := digits[rand.IntN(len(digits))] //nolint:gosec

	return string(d)
}

// Letter returns a single random alphabetic character (a-z, A-Z) as a string.
func Letter() string {
	l := letters[rand.IntN(len(letters))] //nolint:gosec

	return string(l)
}

// Int generates a random integer within the specified inclusive range [min, max].
// If max is less than min, the arguments are swapped to ensure the range is valid.
func Int(from, to int) int {
	if to < from {
		from, to = to, from
	}

	return rand.IntN(to-from+1) + from //nolint:gosec
}

// Element returns a random element from the provided slice.
// If the slice is empty, it returns the zero value for type T.
func Element[T any](slice []T) T {
	if len(slice) == 0 {
		var zero T
		return zero
	}

	return slice[rand.IntN(len(slice))] //nolint:gosec
}

// URL generates a random mocked HTTPS URL string.
func URL() string {
	return fake.URL()
}

// ImagePNG generates an in-memory PNG image and returns its encoded bytes.
// By default, it creates a square image with random size between 100 and 500 pixels.
// Optional arguments allow overriding the dimensions:
//   - no arguments: random square image (w == h)
//   - one argument: square image with given width/height
//   - two arguments: image with explicit width and height
//
// The image content is a simple deterministic color gradient based on pixel
// coordinates, useful for tests, placeholders, or fixtures.
func ImagePNG(whOpt ...int) ([]byte, error) {
	var w, h = Int(100, 500), Int(100, 500) //nolint:mnd
	if len(whOpt) > 0 {
		w = whOpt[0]
		h = w
		if len(whOpt) == 2 { //nolint:mnd
			h = whOpt[1]
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))

	// random base colors (start → end)
	r1, g1, b1 := Int(0, 255), Int(0, 255), Int(0, 255) //nolint:mnd
	r2, g2, b2 := Int(0, 255), Int(0, 255), Int(0, 255) //nolint:mnd

	wf := float64(w - 1)
	hf := float64(h - 1)

	for y := range h {
		ty := float64(y) / hf
		for x := range w {
			tx := float64(x) / wf

			// diagonal gradient factor (0..1)
			t := (tx + ty) * 0.5 //nolint:mnd

			r := uint8(float64(r1)*(1-t) + float64(r2)*t)
			g := uint8(float64(g1)*(1-t) + float64(g2)*t)
			b := uint8(float64(b1)*(1-t) + float64(b2)*t)

			img.Set(x, y, color.RGBA{
				R: r,
				G: g,
				B: b,
				A: 255, //nolint:mnd
			})
		}
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Bool returns a pseudo-random boolean value.
func Bool() bool {
	return rand.IntN(2) == 1 //nolint:gosec,mnd
}

// NilOrValue returns a pointer to val or nil based on the given probability.
// The probability is specified in percent (0–100) and defaults to 50% if omitted.
//
// Examples:
//
//	NilOrValue("hello")       // ~50% chance to return &"hello"
//	NilOrValue("hello", 10)   // 10% chance to return &"hello"
//	NilOrValue("hello", 100)  // always returns &"hello"
//	NilOrValue("hello", 0)    // always returns nil
//
//nolint:mnd
func NilOrValue[T any](val T, probabilityOpt ...int) *T {
	probability := 50
	if len(probabilityOpt) == 1 {
		probability = probabilityOpt[0]
	}

	if probability <= 0 {
		return nil
	}
	if probability >= 100 {
		return &val
	}

	if rand.IntN(100) < probability { //nolint:gosec
		return &val
	}

	return nil
}

// Maybe returns either the provided value or the zero value of T,
// based on the given probability.
func Maybe[T any](val T, probabilityOpt ...int) T {
	var v T
	if NilOrValue(v, probabilityOpt...) != nil {
		return val
	}

	return v
}

// Patch applies an Edit to the input string and returns a patch.
func Patch(input string) (patch string) {
	return app.MakePatch(input, Edit(input))
}

// Edit applies a random modification to the input string that is nicely fit to diff comparison and patch creation.
// The function randomly performs one of six operations:
//   - For multi-word strings (split by spaces): adds/removes entire words
//   - For single-word strings: adds/removes individual characters
//
// If input is empty, it generates a random title first.
//
//nolint:mnd,gosec
func Edit(input string) (out string) {
	if input == "" {
		input = Title()
	}

	parts := strings.Split(input, " ")
	isMultiWord := len(parts) > 1

	switch Int(1, 6) {
	case 1: // add fragment to start
		if isMultiWord {
			parts = append([]string{Title()}, parts...)
		} else {
			parts[0] = Letter() + parts[0]
		}

	case 2: // add fragment to end
		if isMultiWord {
			parts = append(parts, Title())
		} else {
			parts[0] += Letter()
		}
	case 3: // add fragment in the middle
		if isMultiWord {
			idx := rand.IntN(len(parts)-1) + 1
			parts = append(parts[:idx], append([]string{Title()}, parts[idx:]...)...)
		} else {
			word := parts[0]
			pos := 0
			if len(word) > 0 {
				pos = rand.IntN(len(word))
			}
			parts[0] = word[:pos] + Letter() + word[pos:]
		}

	case 4: // remove fragment from start
		if isMultiWord {
			parts = parts[1:]
		} else if len(parts[0]) > 1 {
			parts[0] = parts[0][1:]
		}

	case 5: // remove fragment from end
		if isMultiWord {
			parts = parts[:len(parts)-1]
		} else if len(parts[0]) > 1 {
			parts[0] = parts[0][:len(parts[0])-1]
		}

	case 6: // remove fragment from inside
		if isMultiWord {
			if len(parts) == 2 {
				parts = parts[1:]
			} else {
				idx := rand.IntN(len(parts)-2) + 1
				parts = append(parts[:idx], parts[idx+1:]...)
			}
		} else if len(parts[0]) > 2 {
			pos := rand.IntN(len(parts[0])-2) + 1
			word := parts[0]
			parts[0] = word[:pos] + word[pos+1:]
		}
	}

	return strings.Join(parts, " ")
}
