package main

import (
	"bytes"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/friendsofgo/errors"
	"github.com/sosodev/duration"
	"google.golang.org/api/youtube/v3"
)

var allMdV2 = []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
var mdV2Repl = strings.NewReplacer(func() (out []string) {
	for _, x := range allMdV2 {
		out = append(out, x, "\\"+x)
	}
	return out
}()...)

func EscapeMarkdownV2(s string) string {
	return mdV2Repl.Replace(s)
}

func Text2ExpandableQuote(text string) string {
	result := ""
	splits := strings.Split(text, "\n")
	for i, s := range splits {
		if i == 0 {
			result += "**>" + EscapeMarkdownV2(s) + "\n"
		} else if i == len(splits)-1 {
			result += ">" + EscapeMarkdownV2(s) + "||"
		} else {
			result += ">" + EscapeMarkdownV2(s) + "\n"
		}
	}
	return result
}

type RichMessageOpt struct {
	VideoTitle         string
	VideoUrl           string
	ThumbnailUrl       string
	VideoDescription   string
	VideoDuration      string
	ChannelName        string
	ChannelUrl         string
	AllowedRegion      string
	BlockedRegion      string
	ScheduledStartTime string
	PublishedTime      string
	TimeZone           string
	DefaultLanguage    string
	Localizations      map[string]youtube.VideoLocalization
}

func getUtf16Len(s string) int64 {
	return int64(len(utf16.Encode([]rune(s))))
}

var priorityOrder = map[string]int{
	"zh-Hant": 1,
	"zh-TW":   2,
	"zh-HK":   3,
	"zh-Hans": 4,
	"zh-CN":   5,
	"zh":      6,
	"en-US":   7,
	"en":      8,
}

func getPriority(key string) int {
	if p, exists := priorityOrder[key]; exists {
		return p
	}
	return 99
}

func BuildRichMessage(opt RichMessageOpt) gotgbot.InputRichMessage {
	inputRichBlocks := []gotgbot.InputRichBlock{}

	if opt.VideoTitle != "" {
		inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockSectionHeading{
			Text: gotgbot.RichTextUrl{
				Text: gotgbot.RichTextString(opt.VideoTitle),
				Url:  opt.VideoUrl,
			},
			Size: 3,
		})
	}

	preferedLocale := ytGetPreferedLocale(opt.Localizations, opt.DefaultLanguage)
	if preferedLocale != nil {
		inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockBlockQuotation{
			Blocks: []gotgbot.InputRichBlock{
				gotgbot.InputRichBlockParagraph{
					Text: gotgbot.RichTextString(preferedLocale.Title),
				},
			},
		})
	}

	if opt.ThumbnailUrl != "" {
		inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockPhoto{
			Photo: gotgbot.InputMediaPhoto{
				Media: gotgbot.InputFileByURL(opt.ThumbnailUrl),
			},
		})
	}

	richTextChannel := gotgbot.RichTextArray{}
	if opt.ChannelName != "" {
		richTextChannel = append(richTextChannel, gotgbot.RichTextString("频道: "), gotgbot.RichTextUrl{
			Text: gotgbot.RichTextString(opt.ChannelName),
			Url:  opt.ChannelUrl,
		}, gotgbot.RichTextString("\n"))
	}

	if len(opt.ScheduledStartTime) > 0 {
		richTextChannel = append(richTextChannel, gotgbot.RichTextString("首播时间: "))
		parsedTime, err := time.Parse("2006-01-02T15:04:05Z", opt.ScheduledStartTime)
		if err != nil {
			richTextChannel = append(richTextChannel, gotgbot.RichTextCode{
				Text: gotgbot.RichTextString(opt.ScheduledStartTime),
			})
		} else {
			loc, _ := time.LoadLocation(opt.TimeZone)
			richTextChannel = append(richTextChannel, gotgbot.RichTextDateTime{
				Text:           gotgbot.RichTextString(parsedTime.In(loc).Format("02/01/2006 15:04:05 MST-07")),
				UnixTime:       parsedTime.Unix(),
				DateTimeFormat: "wdT",
			})
		}
		richTextChannel = append(richTextChannel, gotgbot.RichTextString("\n"))
	}

	if len(opt.PublishedTime) > 0 {
		richTextChannel = append(richTextChannel, gotgbot.RichTextString("发布时间: "))
		parsedTime, err := time.Parse("2006-01-02T15:04:05Z", opt.PublishedTime)
		if err != nil {
			richTextChannel = append(richTextChannel, gotgbot.RichTextCode{
				Text: gotgbot.RichTextString(opt.PublishedTime),
			})
		} else {
			loc, _ := time.LoadLocation(opt.TimeZone)
			richTextChannel = append(richTextChannel, gotgbot.RichTextDateTime{
				Text:           gotgbot.RichTextString(parsedTime.In(loc).Format("02/01/2006 15:04:05 MST-07")),
				UnixTime:       parsedTime.Unix(),
				DateTimeFormat: "wdT",
			})
		}
		richTextChannel = append(richTextChannel, gotgbot.RichTextString("\n"))
	}

	if len(opt.VideoDuration) > 0 && opt.VideoDuration != "PT0S" && opt.VideoDuration != "P0D" {
		richTextChannel = append(richTextChannel, gotgbot.RichTextString("时长: "))
		d, err := duration.Parse(opt.VideoDuration)
		if err != nil {
			richTextChannel = append(richTextChannel, gotgbot.RichTextCode{
				Text: gotgbot.RichTextString(opt.VideoDuration),
			})
		} else {
			richTextChannel = append(richTextChannel, gotgbot.RichTextCode{
				Text: gotgbot.RichTextString(d.ToTimeDuration().String()),
			})
		}
	}

	inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockBlockQuotation{
		Blocks: []gotgbot.InputRichBlock{
			gotgbot.InputRichBlockParagraph{
				Text: richTextChannel,
			},
		},
	})

	if len(opt.AllowedRegion) > 0 {
		if len(opt.AllowedRegion) > 144 {
			inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockDetails{
				Summary: gotgbot.RichTextString("允许地区"),
				Blocks: []gotgbot.InputRichBlock{
					gotgbot.InputRichBlockBlockQuotation{
						Blocks: []gotgbot.InputRichBlock{
							gotgbot.InputRichBlockParagraph{
								Text: gotgbot.RichTextCode{
									Text: gotgbot.RichTextString(opt.AllowedRegion),
								},
							},
						},
					},
				},
			})
		} else {
			inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockBlockQuotation{
				Blocks: []gotgbot.InputRichBlock{
					gotgbot.InputRichBlockParagraph{
						Text: gotgbot.RichTextArray{
							gotgbot.RichTextString("允许地区: "),
							gotgbot.RichTextCode{
								Text: gotgbot.RichTextString(opt.AllowedRegion),
							},
						},
					},
				},
			})
		}
	}

	if len(opt.BlockedRegion) > 0 {
		if len(opt.BlockedRegion) > 144 {
			inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockDetails{
				Summary: gotgbot.RichTextString("屏蔽地区"),
				Blocks: []gotgbot.InputRichBlock{
					gotgbot.InputRichBlockBlockQuotation{
						Blocks: []gotgbot.InputRichBlock{
							gotgbot.InputRichBlockParagraph{
								Text: gotgbot.RichTextCode{
									Text: gotgbot.RichTextString(opt.BlockedRegion),
								},
							},
						},
					},
				},
			})
		} else {
			inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockBlockQuotation{
				Blocks: []gotgbot.InputRichBlock{
					gotgbot.InputRichBlockParagraph{
						Text: gotgbot.RichTextArray{
							gotgbot.RichTextString("屏蔽地区: "),
							gotgbot.RichTextCode{
								Text: gotgbot.RichTextString(opt.BlockedRegion),
							},
						},
					},
				},
			})
		}
	}

	if len(opt.VideoDescription) > 0 {
		if strings.Count(opt.VideoDescription, "\n") > 4 {
			inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockDetails{
				Summary: gotgbot.RichTextString("简介"),
				Blocks: []gotgbot.InputRichBlock{
					gotgbot.InputRichBlockBlockQuotation{
						Blocks: []gotgbot.InputRichBlock{
							gotgbot.InputRichBlockParagraph{
								Text: gotgbot.RichTextString(opt.VideoDescription),
							},
						},
					},
				},
				IsOpen: false,
			})
		} else {
			inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockBlockQuotation{
				Blocks: []gotgbot.InputRichBlock{
					gotgbot.InputRichBlockParagraph{
						Text: gotgbot.RichTextString(opt.VideoDescription),
					},
				},
			})
		}
	}

	keys := slices.Collect(maps.Keys(opt.Localizations))
	if len(opt.DefaultLanguage) > 0 {
		keys = slices.DeleteFunc(keys, func(k string) bool {
			return k == opt.DefaultLanguage
		})
	}
	if len(keys) > 0 {
		slices.SortFunc(keys, func(a, b string) int {
			pA, pB := getPriority(a), getPriority(b)
			if pA != pB {
				return pA - pB
			}
			return strings.Compare(a, b)
		})
		richBlockLocale := []gotgbot.InputRichBlock{}
		for _, k := range keys {
			l := opt.Localizations[k]
			richBlockLocale = append(richBlockLocale, gotgbot.InputRichBlockDetails{
				Summary: gotgbot.RichTextString(k),
				Blocks: []gotgbot.InputRichBlock{
					gotgbot.InputRichBlockBlockQuotation{
						Blocks: []gotgbot.InputRichBlock{
							gotgbot.InputRichBlockParagraph{
								Text: gotgbot.RichTextString(l.Title),
							},
						},
					},
					gotgbot.InputRichBlockBlockQuotation{
						Blocks: []gotgbot.InputRichBlock{
							gotgbot.InputRichBlockParagraph{
								Text: gotgbot.RichTextString(l.Description),
							},
						},
					},
				},
			})
		}

		inputRichBlocks = append(inputRichBlocks, gotgbot.InputRichBlockDetails{
			Summary: gotgbot.RichTextString("翻译"),
			Blocks:  richBlockLocale,
		})
	}

	return gotgbot.InputRichMessage{
		Blocks: inputRichBlocks,
	}
}

func GetTimeZone(language string) string {
	switch language {
	case "zh-Hant":
		return "Asia/Taipei"
	case "zh-Hans":
		return "Asia/Shanghai"
	case "ja":
		return "Asia/Tokyo"
	case "ko":
		return "Asia/Seoul"
	default:
		return "UTC"
	}
}

type FileTooLargeError struct{}

func (e *FileTooLargeError) Error() string {
	return "file too large"
}

func downloadToBuffer(url string) (gotgbot.InputFileOrString, error) {
	defaultClient := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := defaultClient.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download file")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to download file")
	}

	if resp.ContentLength > 10*1024*1024 {
		return nil, &FileTooLargeError{}
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}

	fn := "cover.bin"
	switch resp.Header.Get("Content-Type") {
	case "image/jpeg":
		fn = "cover.jpg"
	case "image/png":
		fn = "cover.png"
	}

	return gotgbot.InputFileByReader(fn, buf), nil
}

func truncateByRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

func mtprotoId2chatId(mtprotoId int64) int64 {
	return -(1000000000000 + mtprotoId)
}

func chatId2mtprotoId(chatId int64) int64 {
	return -chatId - 1000000000000
}

func ytGetPreferedLocale(l map[string]youtube.VideoLocalization, d string) *youtube.VideoLocalization {
	preferedLocale := []string{"zh-Hant", "zh-TW", "zh-HK", "zh-Hans", "zh-CN", "zh", "en-US", "en"}

	for _, locale := range preferedLocale {
		if v, ok := l[locale]; ok && locale != d {
			return &v
		}
	}

	return nil
}
