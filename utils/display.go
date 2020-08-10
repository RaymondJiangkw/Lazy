package utils

import (
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/cheggaaa/pb"
)

type Generator struct {
}

func (g *Generator) RepeatedString(s string, length int) string {
	var ret string
	for i := 0; i < length; i++ {
		ret += s
	}
	return ret
}

func (g *Generator) FmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (g *Generator) Flush(length int) string {
	return g.RepeatedString("\r", length) + g.RepeatedString(" ", length) + g.RepeatedString("\r", length)
}

type Display struct {
	generator Generator
}

type SliderOption struct {
	Writer io.Writer
	Prefix string
	// default is 100 * time.Millisecond
	Delay time.Duration
	// Upon received from signal, it stops.
	Signal <-chan struct{}
	Slides string
}

/*
 * @param options SliderOption
 * @param options.Writer io.Writer
 * @param options.prefix string
 * @param options.delay time.Duration (default: 100 * time.Millisecond)
 * @param options.signal <-chan struct{} (must not be nil)
 * @param options.Slides string e.g. `-\|/`
 */
func (d *Display) Slider(options *SliderOption) error {
	if options.Signal == nil || options.Writer == nil {
		return Invalid
	}
	if options.Delay <= time.Duration(0) {
		options.Delay = defaultDelayTime
	}
	for {
		for _, r := range options.Slides {
			select {
			case <-options.Signal:
				return nil
			default:
				fmt.Fprintf(options.Writer, "%s%s%c", d.generator.RepeatedString("\r", len(options.Prefix)+1), options.Prefix, r)
				time.Sleep(options.Delay)
			}
		}
	}
}

/*
 * @param options SliderOption
 * @param options.Writer io.Writer
 * @param options.prefix string
 * @param options.delay time.Duration (default: 100 * time.Millisecond)
 * @param options.signal <-chan struct{} (must not be nil)
 */
func (d *Display) Spinner(options *SliderOption) error {
	options.Slides = `-\|/`
	return d.Slider(options)
}

type ProgressBarOption struct {
	Writer  io.Writer
	Prefix  [][]string
	Postfix [][]string
	Maximum [][]int
	Signal  [][]<-chan struct{}
	Phase   []int
}

/*
 * ProgressBar wrap pb provided by https://github.com/cheggaaa/pb.
 * @param options ProgressBarOption
 * @param options.Writer io.Writer
 * @param options.Prefix [][]string
 * @param options.Postfix [][]string
 * @param options.Maximum [][]int
 * @param options.Signal [][]<-chan struct{}
 * @param options.Phase []int
 */
func (d *Display) ProgressBar(options *ProgressBarOption) (<-chan struct{}, error) {
	if len(options.Prefix) != len(options.Maximum) || len(options.Maximum) != len(options.Signal) || len(options.Signal) != len(options.Phase) || len(options.Prefix) != len(options.Postfix) {
		return nil, Invalid
	}
	var cnt = len(options.Phase)
	// Check Validity
	if options.Writer == nil {
		return nil, Invalid
	}
	for i := 0; i < cnt; i++ {
		if options.Phase[i] < 1 || len(options.Prefix[i]) != options.Phase[i] || len(options.Postfix[i]) != options.Phase[i] {
			return nil, Invalid
		}
		for j := 0; j < options.Phase[i]; j++ {
			if options.Maximum[i][j] < 0 {
				return nil, Invalid
			}
		}
	}
	finish := make(chan struct{})
	go func() {
		defer func() {
			finish <- struct{}{}
		}()
		var progressBars []*pb.ProgressBar
		for i := 0; i < cnt; i++ {
			bar := pb.New(options.Maximum[i][0])
			progressBars = append(progressBars, bar)
		}
		pool, err := pb.StartPool(progressBars...)
		if err != nil {
			// NOTICE: Omit Error here! Since it is inside another thread.
			return
		}
		var wg sync.WaitGroup
		for i, bar := range progressBars {
			wg.Add(1)
			go func(i int, bar *pb.ProgressBar) {
				defer wg.Done()
				for p := 0; p < options.Phase[i]; p++ {
					bar.Set(0)
					bar.SetTotal(options.Maximum[i][p])
					bar.Prefix(options.Prefix[i][p])
					bar.Postfix(options.Postfix[i][p])
					for {
						// fmt.Println(bar.IsFinished(), bar.Get())
						if bar.Get() == int64(options.Maximum[i][p]) {
							break
						}
						if options.Signal[i][p] != nil {
							<-options.Signal[i][p]
							bar.Increment()
						} else {
							bar.Finish()
						}
					}
				}
			}(i, bar)
		}
		wg.Wait()
		pool.Stop()
	}()
	return finish, nil
}

func (d *Display) EasyProgress(writer io.Writer, prefix string, postfix string, maximum int, signal <-chan struct{}) (<-chan struct{}, error) {
	if maximum < 0 || signal == nil {
		return nil, Invalid
	}
	finish := make(chan struct{})
	go func() {
		defer func() {
			finish <- struct{}{}
		}()
		var text string
		var cnt int
		for {
			fmt.Fprintf(writer, "%s", d.generator.Flush(len(text)))
			text = prefix + "( " + strconv.Itoa(cnt) + "/" + strconv.Itoa(maximum) + " )" + postfix
			fmt.Fprintf(writer, "%s", text)
			if cnt == maximum {
				break
			}
			<-signal
			cnt++
		}
		fmt.Fprintf(writer, "%s", d.generator.Flush(len(text)))
	}()
	return finish, nil
}

func (d *Display) TemporaryText(writer io.Writer, text string, signal <-chan struct{}) <-chan struct{} {
	finish := make(chan struct{})
	go func() {
		defer func() {
			finish <- struct{}{}
		}()
		fmt.Fprintf(writer, "%s", text)
		<-signal
		fmt.Fprintf(writer, "%s", d.generator.Flush(len(text)))
	}()
	return finish
}
