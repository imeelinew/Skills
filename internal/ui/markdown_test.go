package ui

import (
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestMarkdownUsesNativeBodyAndRGBColors(t *testing.T) {
	for _, dark := range []bool{false, true} {
		body, err := renderMarkdown("Plain body text.", 60, dark)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(body, "38;") || strings.Contains(body, "48;") {
			t.Fatalf("body must inherit the terminal foreground: %q", body)
		}
		heading, err := renderMarkdown("# Heading", 60, dark)
		if err != nil || strings.Contains(heading, "38;") || strings.Contains(heading, "48;") {
			t.Fatalf("H1 must use the terminal foreground, without yellow or a backplate: %q", heading)
		}
		rich, err := renderMarkdown("# Title\n\n## Section\n\n`code` [link](https://example.com)\n\n```go\n// Comment\nfunc main() { fmt.Println(42, \"hello\") }\n```", 60, dark)
		if err != nil {
			t.Fatal(err)
		}
		if regexp.MustCompile(`\x1b\[[0-9;]*(?:38|48);5;`).MatchString(rich) {
			t.Fatalf("Markdown emitted palette indices: %q", rich)
		}
		if !strings.Contains(rich, "38;2;") {
			t.Fatal("Markdown accents were not rendered as RGB")
		}
	}
}

func TestSyntaxColorsFollowThemeChanges(t *testing.T) {
	for _, dark := range []bool{true, false, true, false} {
		t.Run(strconv.FormatBool(dark), func(t *testing.T) {
			t.Parallel()
			out, err := renderMarkdown("```go\nfmt.Println(42)\n```", 60, dark)
			if err != nil {
				t.Fatal(err)
			}
			want := "38;2;37;44;56"
			if dark {
				want = "38;2;216;222;233"
			}
			if !strings.Contains(out, want) {
				t.Fatalf("dark=%v: syntax palette was reused from another theme: %q", dark, out)
			}
		})
	}
}

func TestMarkdownColorsHaveReadableContrast(t *testing.T) {
	for _, dark := range []bool{false, true} {
		style := markdownStyle(dark)
		background := "#FBF9F5"
		if dark {
			background = "#171B22"
		}
		var check func(reflect.Value, string)
		check = func(value reflect.Value, bg string) {
			if value.Kind() == reflect.Pointer {
				if !value.IsNil() {
					check(value.Elem(), bg)
				}
				return
			}
			if value.Kind() != reflect.Struct {
				return
			}
			if c := value.FieldByName("BackgroundColor"); c.IsValid() && !c.IsNil() {
				bg = c.Elem().String()
			}
			if c := value.FieldByName("Color"); c.IsValid() && !c.IsNil() {
				fg := c.Elem().String()
				if fg != "" {
					if !strings.HasPrefix(fg, "#") {
						t.Fatalf("non-RGB color %q", fg)
					}
					if ratio := contrast(fg, bg); ratio < 4.5 {
						t.Errorf("dark=%v: %s against %s has contrast %.2f", dark, fg, bg, ratio)
					}
				}
			}
			for i := 0; i < value.NumField(); i++ {
				check(value.Field(i), bg)
			}
		}
		check(reflect.ValueOf(style), background)
	}
}

func contrast(a, b string) float64 {
	luminance := func(hex string) float64 {
		var channels [3]float64
		for i := range channels {
			v, _ := strconv.ParseUint(hex[1+i*2:3+i*2], 16, 8)
			c := float64(v) / 255
			if c <= 0.04045 {
				channels[i] = c / 12.92
			} else {
				channels[i] = math.Pow((c+0.055)/1.055, 2.4)
			}
		}
		return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
	}
	x, y := luminance(a), luminance(b)
	return (max(x, y) + 0.05) / (min(x, y) + 0.05)
}
