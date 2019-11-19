package main

import (
    "fmt"
    "os"
    "io/ioutil"
    "hash/fnv"
    "github.com/kavorite/vader"
)

func fck(what string, cause error) {
    if cause == nil {
        return
    }
    h := fnv.New32a()
    h.Write([]byte(what))
    h.Write([]byte(cause.Error()))
    code := int(h.Sum32() & 0xffff)
    if cause != nil {
        fmt.Fprintf(os.Stderr, "%s: %s (0×%x)\n", what, cause, code)
        os.Exit(code)
    }
}

func main() {
    src, err := ioutil.ReadAll(os.Stdin)
    fck("read stdin", err)
    D := vader.ParseText(string(src))
    fmt.Printf("%+v\n", D.PolarityScores())
}
