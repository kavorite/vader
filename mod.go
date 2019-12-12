package vader

import (
    "math"
    "strings"
    "unicode"

    "golang.org/x/text/transform"
    "golang.org/x/text/unicode/norm"
)

func strip(src string) string {
    strip := func(r rune) bool {
        return unicode.Is(unicode.Mn, r)
    }
    t := transform.Chain(norm.NFD, transform.RemoveFunc(strip), norm.NFC)
    stripped, _, _ := transform.String(t, src)
    return stripped
}

func tokenize(src string) (T []string) {
    D := strings.Fields(strip(src))
    T = make([]string, len(D), len(D))
    for i, t := range D {
        if len(t) == 1 {
            continue
        }
        T[i] = strings.Trim(t, punctuation)
    }
    return
}

const (
    // positive term valence
    boostIncr = 0.293
    // negative term valence
    boostDecr = -0.293

    // caps for emphasis
    capsIncr = 0.733

    // negations
    negationCoeff = -0.740
    // quotations
    qMarkIncr = 0.180
    // exclamation points
    bangIncr = 0.292

    maxBangc     = 4
    maxQmarkc    = 3
    maxQmarkIncr = 0.96
    normAlpha    = 15.0

    punctuation = "[!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~]"
)

type BOW map[string]struct{}

func Bag(words ...string) (bag BOW) {
    bag = make(BOW, len(words))
    for _, t := range words {
        bag[t] = struct{}{}
    }
    return
}

func (bag BOW) Has(t string) bool {
    _, ok := bag[t]
    return ok
}

func (bag BOW) ContainsAny(of ...string) bool {
    for _, t := range of {
        if bag.Has(t) {
            return true
        }
    }
    return false
}

var negations = Bag(
    "aint", "arent", "cannot", "cant", "couldnt", "darent", "didnt",
    "doesnt", "ain't", "aren't", "can't", "couldn't", "daren't", "didn't",
    "doesn't", "dont", "hadnt", "hasnt", "havent", "isnt", "mightnt",
    "mustnt", "neither", "don't", "hadn't", "hasn't", "haven't", "isn't",
    "mightn't", "mustn't", "neednt", "needn't", "never", "none", "nope",
    "nor", "not", "nothing", "nowhere", "oughtnt", "shant", "shouldnt",
    "uhuh", "wasnt", "werent", "oughtn't", "shan't", "shouldn't", "uh-uh",
    "wasn't", "weren't", "without", "wont", "wouldnt", "won't", "wouldn't",
    "rarely", "seldom", "despite",
)

var positives = Bag(
    "absolutely", "amazingly", "awfully", "completely", "considerably",
    "decidedly", "deeply", "effing", "enormously", "entirely", "especially",
    "exceptionally", "extremely", "fabulously", "flipping", "flippin",
    "fricking", "frickin", "frigging", "friggin", "fully", "fucking", "fuckin",
    "greatly", "hella", "highly", "hugely", "incredibly", "intensely",
    "majorly", "more", "most", "particularly", "purely", "quite", "really",
    "remarkably", "so", "substantially", "thoroughly", "totally",
    "tremendously", "uber", "unbelievably", "unusually", "utterly", "very",
)

var negatives = Bag(
    "almost", "barely", "hardly", "just enough", "kind of", "kinda",
    "kindof", "kind-of", "less", "little", "marginally",
    "occasionally", "partly", "scarcely", "slightly", "somewhat",
    "sort of", "sorta", "sortof", "sort-of",
)

var spCaseIdioms = map[string]float64{
    "the shit":      3.0,
    "the bomb":      3.0,
    "bad ass":       1.5,
    "yeah right":    -2.0,
    "kiss of death": -1.5,
}

func boost(t string) float64 {
    if positives.Has(t) {
        return boostIncr
    } else if negatives.Has(t) {
        return boostDecr
    }
    return 0
}

func appendEmojiDescs(src string) string {
    var B, Q strings.Builder
    for _, r := range src {
        if description, ok := emojiLexicon[string(r)]; ok {
            Q.WriteByte(' ')
            Q.WriteString(description)
        } else {
            B.WriteRune(r)
        }
    }
    B.WriteString(Q.String())
    return B.String()
}

type Doc struct {
    tokens    []string
    ltokens   []string
    mixedCaps bool
    punctAmp  float64
}

func ParseText(raw string) (D Doc) {
    src := appendEmojiDescs(raw)
    D.tokens = tokenize(src)
    D.ltokens = make([]string, len(D.tokens))
    var (
        hasCaps    = false
        hasNonCaps = false
        qmarkc     = 0
        bangc      = 0
    )

    for i, t := range D.tokens {
        D.ltokens[i] = strings.ToLower(t)
        if strings.ToUpper(t) == t {
            hasCaps = true
            if hasNonCaps {
                D.mixedCaps = true
            }
        } else {
            hasNonCaps = true
            if hasCaps {
                D.mixedCaps = true
            }
        }
    }

    for _, c := range src {
        switch c {
        case '!':
            bangc++
        case '?':
            qmarkc++
        }
    }

    if bangc > maxBangc {
        bangc = maxBangc
    }
    D.punctAmp = bangIncr * float64(bangc)
    if qmarkc > maxQmarkc {
        D.punctAmp += maxQmarkIncr
    } else {
        D.punctAmp += qMarkIncr * float64(qmarkc)
    }
    return
}

type PolarityScores struct {
    Positive, Negative, Neutral, Compound float64
}

func negatesp(t string) bool {
    t = strings.ToLower(t)
    if _, ok := negations[t]; ok {
        return true
    }
    return strings.Contains(t, "n't")
}

func scalarIncDec(t string, valence float64, mixedCaps bool) (s float64) {
    s = boost(t)
    if s == 0 {
        return
    }
    if valence < 0 {
        s *= -1
    }
    if len(t) > 1 && strings.ToUpper(t) == t && mixedCaps {
        if valence > 0 {
            s += capsIncr
        } else {
            s -= capsIncr
        }
    }
    return
}

func negationCheck(valence float64, ltokens []string, j, i int) float64 {
    switch j {
    case 0:
        if negatesp(ltokens[i-1]) {
            valence *= negationCoeff
        }
    case 1:
        if ltokens[i-2] == "never" && Bag("so", "this").Has(ltokens[i-1]) {
            valence *= 1.25
        } else if ltokens[i-2] == "without" && ltokens[i-1] == "doubt" {
            break
        } else if negatesp(ltokens[i-2]) {
            valence *= negationCoeff
        }
    case 2:
        if ltokens[i-3] == "never" && Bag("so", "this").ContainsAny(ltokens[i-2:i]...) {
            valence *= 1.25
        } else if ltokens[i-3] == "without" && Bag(ltokens[i-2:i]...).Has("doubt") {
            valence *= 1.0
        } else if negatesp(ltokens[i-3]) {
            valence *= negationCoeff
        }
    }
    return valence
}

func spIdiomsCheck(valence float64, ltokens []string, i int) float64 {
    if i < 3 {
        return valence
    }

    k := i + 1
    if len(ltokens)-1 > i {
        k = i + 3
        if k > len(ltokens) {
            k = len(ltokens)
        }
    }

    context := strings.Join(ltokens[(i-3):k], " ")
    for idiom, v := range spCaseIdioms {
        if strings.Contains(context, idiom) {
            valence = v
            break
        }
    }

    for j := i - 3; j < i; j++ {
        valence += boost(ltokens[j])
    }
    return valence
}

func butCheck(ltokens []string, sentiments []float64) {
    i := 0
    for ; i < len(ltokens); i++ {
        if ltokens[i] == "but" {
            break
        } else if i == len(ltokens) {
            return
        }
    }
    for j := 0; j < i-1; j++ {
        sentiments[j] *= 0.5
    }
    for j := i + 1; j < len(sentiments)-1; j++ {
        sentiments[j] *= 1.5
    }
}

func leastCheck(valence float64, ltokens []string, i int) float64 {
    if i > 1 && ltokens[i-1] == "least" && !Bag("very", "at").Has(ltokens[i-2]) {
        valence *= negationCoeff
    } else if i > 0 && ltokens[i-1] == "least" {
        valence *= negationCoeff
    }
    return valence
}

func (D Doc) sentimentValence(i int) (valence float64) {
    valence = lexicon.Valence(D.ltokens[i])
    if valence == 0 {
        return
    }
    if D.ltokens[i] == "no" && lexicon.Has(D.ltokens[i+1]) {
        valence = 0
    }

    if (i > 0 && D.ltokens[i-1] == "no") ||
    (i > 1 && D.ltokens[i-2] == "no") ||
    (i > 2 && D.ltokens[i-3] == "no" &&
    Bag("nor", "or").Has(D.ltokens[i-1])) {
        valence *= negationCoeff
    }

    if D.mixedCaps &&
    len(D.tokens[i]) > 1 &&
    strings.ToUpper(D.ltokens[i]) == D.ltokens[i] {
        if valence > 0 {
            valence += capsIncr
        } else {
            valence -= capsIncr
        }
    }

    for j := 0; j < 3; j++ {
        if i > j && !lexicon.Has(D.ltokens[i-j-1]) {
            s := scalarIncDec(D.tokens[i-j-1], valence, D.mixedCaps)
            s *= 1.0 - 0.05*float64(j)
            valence += s
            valence = negationCheck(valence, D.ltokens, j, i)
            if j == 2 {
                valence = spIdiomsCheck(valence, D.ltokens, i)
            }
        }
    }
    valence = leastCheck(valence, D.ltokens, i)
    return
}

func normalizeScore(score float64) (norm float64) {
    norm = score / math.Sqrt(score*score + normAlpha)
    if norm < -1 {
        norm = -1
    } else if norm > 1 {
        norm = 1
    }
    return
}

func getTotalSentiment(sentiments []float64, punctAmp float64) PolarityScores {
    if len(sentiments) == 0 {
        return PolarityScores{}
    }
    var sigma float64
    for _, s := range sentiments {
        sigma += s
    }
    if sigma > 0 {
        sigma += punctAmp
    } else {
        sigma -= punctAmp
    }

    var positive, negative float64
    neutralc := 0
    for _, x := range sentiments {
        if x > 0 {
            positive += x + 1
        }
        if x < 0 {
            negative += x - 1
        }
        if x == 0 {
            neutralc++
        }
    }

    if positive > math.Abs(negative) {
        negative += punctAmp
    } else {
        negative -= punctAmp
    }

    total := positive + math.Abs(negative) + float64(neutralc)

    return PolarityScores{
        Positive: math.Round(100 * positive / total) / 100,
        Negative: math.Round(100 * negative / total) / 100,
        Neutral:  math.Round(100 * float64(neutralc) / total) / 100,
        Compound: math.Round(100 * normalizeScore(sigma)) / 100,
    }
}

func (D Doc) PolarityScores() PolarityScores {
    sentiments := make([]float64, len(D.ltokens))
    for i, t := range D.ltokens {
        if boost(t) != 0 {
            continue
        } else if i < len(D.ltokens)-1 &&
        t == "kind" && D.ltokens[i+1] == "of" {
            continue
        } else {
            sentiments[i] = D.sentimentValence(i)
        }
    }
    butCheck(D.ltokens, sentiments)
    return getTotalSentiment(sentiments, D.punctAmp)
}
