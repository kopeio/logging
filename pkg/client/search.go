package client

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"io"
	"kope.io/klog/pkg/proto"
	"strconv"
	"strings"
	"time"
)

const (
	OutputFormatRaw      = "raw"
	OutputFormatDescribe = "describe"
)

type SearchOptions struct {
	Output string
}

func RunSearch(f Factory, out io.Writer, args []string, o *SearchOptions) error {
	request := &proto.SearchRequest{}

	var formatter func(commonFields *proto.Fields, items []*proto.SearchResult, out io.Writer) error
	switch o.Output {
	case OutputFormatRaw:
		formatter = formatRaw

	case OutputFormatDescribe:
		formatter = formatDescribe

	default:
		return fmt.Errorf("unknown output format %q", o.Output)
	}

	if len(args) > 0 {
		for _, arg := range args {
			// TODO: build a parser properly!
			if strings.Contains(arg, "!=") {
				i := strings.Index(arg, "!=")
				request.FieldFilters = append(request.FieldFilters, &proto.FieldFilter{
					Key:   arg[0:i],
					Value: arg[i+2:],
					Op:    proto.FieldFilterOperator_NOT_EQ,
				})
			} else if strings.Contains(arg, "=") {
				tokens := strings.SplitN(arg, "=", 2)
				key := tokens[0]
				value := tokens[1]
				if key == "age" {
					d, err := parseDurationExpression(value)
					if err != nil {
						return err
					}
					// TODO: Need to sync times somehow
					ts := time.Now().Add(-d)
					request.FieldFilters = append(request.FieldFilters, &proto.FieldFilter{
						Key:   "@timestamp",
						Value: ts.Format(time.RFC3339Nano),
						Op:    proto.FieldFilterOperator_GTE,
					})
				} else {
					request.FieldFilters = append(request.FieldFilters, &proto.FieldFilter{
						Key:   key,
						Value: value,
						Op:    proto.FieldFilterOperator_EQ,
					})
				}
			} else {
				// substring match
				if request.Contains != "" {
					return fmt.Errorf("multiple search not yet implemented")
				}
				request.Contains = arg
			}
		}
	}

	glog.V(2).Infof("query: %v", request)
	client, err := f.LogServerClient()
	if err != nil {
		return err
	}

	// TODO: What is the right context?
	ctx := context.Background()

	stream, err := client.Search(ctx, request)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading from server: %v", err)
		}

		if err := formatter(in.CommonFields, in.Items, out); err != nil {
			return fmt.Errorf("error writing results: %v", err)
		}
	}

	return nil
}

func parseDurationExpression(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	units := time.Minute
	if strings.HasSuffix(s, "s") {
		units = time.Second
		s = s[0 : len(s)-1]
	} else if strings.HasSuffix(s, "m") {
		units = time.Minute
		s = s[0 : len(s)-1]
	} else if strings.HasSuffix(s, "h") {
		units = time.Hour
		s = s[0 : len(s)-1]
	} else if strings.HasSuffix(s, "d") {
		units = time.Hour * 24
		s = s[0 : len(s)-1]
	} else if strings.HasSuffix(s, "w") {
		units = time.Hour * 24 * 7
		s = s[0 : len(s)-1]
	}

	number, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("cannot parse %q as duration", s)
	}

	d := time.Duration(number) * units

	return d, nil
}

func formatRaw(commonFields *proto.Fields, items []*proto.SearchResult, out io.Writer) error {
	var b bytes.Buffer
	for _, item := range items {
		if item.Fields != nil {
			for _, f := range item.Fields.Fields {
				if f.Key != "log" {
					continue
				}
				b.WriteString(strings.TrimSuffix(f.Value, "\n"))
				b.WriteString("\n")
			}
		}
	}

	_, err := b.WriteTo(out)
	return err
}

func formatDescribe(commonFields *proto.Fields, items []*proto.SearchResult, out io.Writer) error {
	var b bytes.Buffer
	for _, item := range items {
		b.WriteString("\n-----------\n")
		t := ""
		if item.Timestamp != 0 {
			seconds := int64(item.Timestamp / 1E9)
			nanos := int64(item.Timestamp % 1E9)
			ts := time.Unix(seconds, nanos)
			t = ts.Format(time.RFC3339Nano)
			b.WriteString("time\t")
			b.WriteString(t)
			b.WriteString("\n")
		}
		if item.Fields != nil {
			for _, f := range item.Fields.Fields {
				if f.Key != "log" {
					continue
				}
				b.WriteString(f.Key)
				b.WriteString("\t")
				b.WriteString(strings.TrimSuffix(f.Value, "\n"))
				b.WriteString("\n")
			}

			for _, f := range item.Fields.Fields {
				if f.Key == "log" {
					continue
				}
				b.WriteString(f.Key)
				b.WriteString("\t")
				b.WriteString(f.Value)
				b.WriteString("\n")
			}
		}
		if commonFields != nil {
			for _, f := range commonFields.Fields {
				b.WriteString(f.Key)
				b.WriteString("\t")
				b.WriteString(f.Value)
				b.WriteString("\n")
			}
		}

	}

	_, err := b.WriteTo(out)
	return err
}
