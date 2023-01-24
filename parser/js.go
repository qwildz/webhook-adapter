package parser

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"runtime"

	"github.com/jackc/puddle/v2"
	v8 "rogchap.com/v8go"
)

var jsPool = newJSParser()
var parsedJS = make(map[uint]*v8.CompilerCachedData)

func newJSParser() *puddle.Pool[*v8.Context] {
	maxPoolSize := int32(runtime.NumCPU())
	constructor := func(ctx context.Context) (*v8.Context, error) {
		iso := v8.NewIsolate()
		global := v8.NewObjectTemplate(iso)

		fetchfn := v8.NewFunctionTemplate(iso, func(info *v8.FunctionCallbackInfo) *v8.Value {
			args := info.Args()
			url := args[0].String()

			resolver, _ := v8.NewPromiseResolver(info.Context())

			go func() {
				res, err := http.Get(url)

				if err != nil {
					errInfo, _ := v8.NewValue(iso, err)
					resolver.Reject(errInfo)
					return
				}

				body, err := io.ReadAll(res.Body)

				if err != nil {
					errInfo, _ := v8.NewValue(iso, err)
					resolver.Reject(errInfo)
					return
				}

				val, err := v8.NewValue(iso, string(body))

				if err != nil {
					errInfo, _ := v8.NewValue(iso, err)
					resolver.Reject(errInfo)
					return
				}

				resolver.Resolve(val)
			}()
			return resolver.GetPromise().Value
		})

		global.Set("fetch", fetchfn, v8.ReadOnly)
		return v8.NewContext(iso, global), nil
	}
	destructor := func(value *v8.Context) {
		value.Close()
		value.Isolate().Dispose()
	}

	pool, err := puddle.NewPool(&puddle.Config[*v8.Context]{
		Constructor: constructor,
		Destructor:  destructor,
		MaxSize:     maxPoolSize,
	})

	if err != nil {
		log.Fatalln(err)
	}

	return pool
}

func RunJS(ctx context.Context, id uint, script, payload string) (string, error) {
	res, _ := jsPool.Acquire(ctx)
	defer res.Release()

	js := res.Value()

	if compiled, _ := parsedJS[id]; true {
		fn, err := js.Isolate().CompileUnboundScript(script, "", v8.CompileOptions{})

		if err != nil {
			log.Fatalln(err)
		}

		_, err = fn.Run(js)

		if err != nil {
			log.Fatalln(err)
		}

		parsedJS[id] = fn.CreateCodeCache()
	} else {
		_, err := js.Isolate().CompileUnboundScript(script, "", v8.CompileOptions{CachedData: compiled})
		if err != nil {
			log.Fatalln(err)
		}
	}

	js.Global().Set("payload", payload)
	val, _ := js.RunScript("transform(payload).then((data) => data)", "")
	prom, _ := val.AsPromise()

	successCb := func(info *v8.FunctionCallbackInfo) *v8.Value {
		return nil
	}

	failCb := func(info *v8.FunctionCallbackInfo) *v8.Value {
		return nil
	}

	prom.Then(successCb, failCb)

	// wait for the promise to resolve
	for prom.State() == v8.Pending {
		continue
	}

	result, err := json.Marshal(prom.Result())

	if err != nil {
		return "", err
	}

	return string(result), nil
}
