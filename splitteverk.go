package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/knakk/rdf"
	"github.com/knakk/sparql"
)

var templ = template.Must(template.New("index").Parse(`
<!DOCTYPE HTML>
<html dir="ltr" lang="no">
	<head>
		<title>Automatisk endring av egenskaper på splittede verk</title>
		<meta http-equiv="Content-Type" content="text/html;charset=utf-8">
		<meta name="viewport" content="initial-scale=1.0">
		<style>
			*         { box-sizing: border-box }
			html      { font-family: Arial,sans-serif; line-height:1.15;
				        -ms-text-size-adjust: 100%;-webkit-text-size-adjust: 100% }
			body      { margin: 0; padding: 0; background: #eee; }
			main      { margin: 0; padding: 2em; width: 50em; margin: auto; font-size: 150%; background: #fff;}
			a, a:visited { text-decoration: none; color: navy;  }
			textarea { width: 100%; margin-top:1em }

			.candidate         { clear:both; margin: 2em 0; padding: 0.2em 0 ;}
			.candidate:hover   {  }
			.candidate-keep    { display: inline-block; width: 2em; float: left;}
			.candidate-title   { cursor: pointer; display: inline-block; float: left; }
			.candidate-details { clear: both; font-family: monospace; margin-top: 2em; }
			.candidate-details:target { display: block; }
			.candidate-from, .candidate-to { width: 40%; float: left; font-size: 60%;}
			.candidate-arrow  { width: 10%; float: left; padding: 2em; }
			.hidden { display: none}

			.queries { clear: both; padding:1em 0;}
			.queries button { font-size: 1em; padding: 0.5em;}
		</style>
	</head>
	<body>
		<main>
			<h2>Automatisk endring av egenskaper på splittede verk</h2>
			<p>Fant {{len .}} kandidater til behandling:</p>
			<div class="candidates">
				{{range .}}
				<div class="candidate" id="{{.ID}}">
					<div class="candidate-keep"><input type="checkbox" value="{{.SPARQL}}" /></div>
					<div class="candidate-title">
						<strong>{{.Title}} </strong>
					</div>
					<div class="candidate-details" >
						<div class="candidate-from">
							{{range $k, $v := .From}}
							<strong>{{$k}}</strong>: {{$v}}<br/>
							{{end}}
						</div>
						<div class="candidate-arrow">⟹</div>
						<div class="candidate-to">
							{{range $k, $v := .To}}
							<strong>{{$k}}</strong>: {{$v}}<br/>
							{{end}}
						</div>
					</div>
				</div>
				{{end}}

			</div>
			<div class="queries">
				<button id="select-all">Velg alle</button> <button id="show-queries">Vis SPARQL spørringer for å oppdatere valgte verk</button><br/>
				<form action="/update" method="POST">
					<textarea name="queries" rows="10" id="selected-queries" readonly="true"></textarea><br/>
					<textarea name="works" rows="10" id="updated-works" style="display:none"></textarea><br/>
				<button type="submit" id="launch-rocket"><strong>⚠ Oppdatér produksjonsdata og reindekser valgte verk ⚠</strong></button>
				</form>
			</div>
		</main>
		<script>
			function toggleChecked(elem) {
				elem.checked = !elem.checked;
			}
			var candidates = document.querySelectorAll(".candidate-title");
			for (var candidate of candidates) {
			    candidate.addEventListener('click', function(event) {
			        toggleChecked(event.target.parentNode.parentNode.querySelector("input"))
			    });
			}

			document.getElementById("show-queries").addEventListener("click", function(event) {
				var result = "";
				var uris = [];
				var inputs = document.querySelectorAll("input[type=checkbox]");
				for (var input of inputs) {
					if (input.checked) {
						result += "\n";
						result += input.value;
						uris.push(input.parentNode.parentNode.id);
					}
				}
				document.getElementById("selected-queries").value = result;
				document.getElementById("updated-works").value = uris.join("\n");
			})

			document.getElementById("select-all").addEventListener("click", function(event) {
				var inputs = document.querySelectorAll("input[type=checkbox]");
				for (var input of inputs) {
					input.checked = !input.checked
				}
			})

		</script>
	</body>
</html>
`))

const queries = `
# tag: candidateWorks
PREFIX : <http://data.deichman.no/ontology#>

SELECT DISTINCT ?prodWork
FROM <https://katalog.deichman.no>
WHERE {
	?prodWork <http://migration.deichman.no/splitFrom> ?fromWork .
}

# tag: prodWork
PREFIX : <http://data.deichman.no/ontology#>

CONSTRUCT {
 <{{.URI}}> <mainTitle> ?mainTitle ;
          <subtitle> ?subtitle ;
          <partTitle> ?partTitle ;
          <partNumber> ?partNumber ;
          :subject ?subject ;
          <subjectLabel> ?subjectLabel ;
          :genre ?genre ;
          <genreLabel> ?genreLabel ;
          <classificationLabel> ?classificationLabel ;
          :audience ?audience ;
          :literaryForm ?litform ;
          :hasWorkType ?worktype ;
          :hasCompositionType ?ctype ;
          <comptype> ?comptype .
}
FROM <https://katalog.deichman.no>
WHERE {
      { <{{.URI}}> :mainTitle ?mainTitle }
UNION { <{{.URI}}> :partTitle ?partTitle }
UNION { <{{.URI}}> :subtitle ?subtitle }
UNION { <{{.URI}}> :partNumber ?partNumber }
UNION { <{{.URI}}> :subject ?subject . ?subject :prefLabel|:name|:mainTitle ?subjectLabel }
UNION { <{{.URI}}> :genre ?genre . ?genre :prefLabel ?genreLabel }
UNION { <{{.URI}}> :audience ?audience }
UNION { <{{.URI}}> :literaryForm ?litform }
UNION { <{{.URI}}> :hasWorkType ?worktype }
UNION { <{{.URI}}> :hasCompositionType ?ctype . ?ctype :prefLabel ?comptype }
UNION { <{{.URI}}> :hasClassification [ :hasClassificationNumber ?classificationLabel ] }
}

# tag: migWork
PREFIX : <http://data.deichman.no/ontology#>

CONSTRUCT {
 <{{.URI}}> <mainTitle> ?mainTitle ;
          <subtitle> ?subtitle ;
          <partTitle> ?partTitle ;
          <partNumber> ?partNumber ;
          <recordId> ?recordId ;
          :subject ?subject ;
          <subjectLabel> ?subjectLabel ;
          :genre ?genre ;
          <genreLabel> ?genreLabel ;
          <classificationLabel> ?classificationLabel ;
          :audience ?audience ;
          :literaryForm ?litform ;
          :hasWorkType ?worktype ;
          :hasCompositionType ?ctype ;
          <comptype> ?comptype ;
          <classNumberAndSource> ?classNumberAndSource .
}
FROM <migration>
FROM NAMED <https://katalog.deichman.no>
WHERE {
	GRAPH <https://katalog.deichman.no> {
					?p :publicationOf <{{.URI}}> ;
						:recordId ?recordId ;
						:mainTitle ?mainTitle .
		OPTIONAL { <{{.URI}}> :partTitle ?partTitle }
		OPTIONAL { <{{.URI}}> :subtitle ?subtitle }
		OPTIONAL { <{{.URI}}> :partNumber ?partNumber }
	}
	      { ?p :subject ?subject . ?subject :prefLabel|:name|:mainTitle ?subjectLabel }
	UNION { ?p :genre ?genre . ?genre :prefLabel ?genreLabel }
	UNION { ?p :audience ?audience }
	UNION { ?p :literaryForm ?litform  }
	UNION { ?p :hasWorkType ?worktype }
	UNION { ?p :hasCompositionType ?ctype . ?ctype :prefLabel ?comptype }
	UNION { ?p :hasClassification [ :hasClassificationNumber ?classificationLabel ] }
	UNION { ?p :hasClassification ?classEntry . ?classEntry :hasClassificationNumber ?classNumber . OPTIONAL { ?classEntry :hasClassificationSource ?classSource } BIND(IF(BOUND(?classSource), CONCAT(?classNumber, "____", ?classSource), ?classNumber) AS ?classNumberAndSource) }
}

`

var labels = map[rdf.IRI]string{
	mustURI("mainTitle"):                                     "tittel",
	mustURI("subtitle"):                                      "tittel",
	mustURI("partTitle"):                                     "tittel",
	mustURI("partNumber"):                                    "tittel",
	mustURI("subjectLabel"):                                  "emne",
	mustURI("genreLabel"):                                    "sjanger",
	mustURI("recordId"):                                      "tittelnr",
	mustURI("classificationLabel"):                           "klassifikasjon",
	mustURI("comptype"):                                      "komposisjonstype",
	mustURI("http://data.deichman.no/ontology#audience"):     "målgruppe",
	mustURI("http://data.deichman.no/ontology#hasWorkType"):  "verkstype",
	mustURI("http://data.deichman.no/ontology#literaryForm"): "litterær form",
}

func mustURI(s string) rdf.IRI {
	uri, err := rdf.NewIRI(s)
	if err != nil {
		panic(err)
	}
	return uri
}

type Main struct {
	queries  sparql.Bank
	virtuoso *sparql.Repo
}

func newMain() (*Main, error) {
	repo, err := sparql.NewRepo("http://virtuoso:8890/sparql")
	if err != nil {
		return nil, err
	}
	return &Main{
		queries:  sparql.LoadBank(bytes.NewBufferString(queries)),
		virtuoso: repo,
	}, nil
}

func (m *Main) run() ([]workDiff, error) {
	log.Println("Finner splittet verk kandidater")

	q, err := m.queries.Prepare("candidateWorks", nil)
	if err != nil {
		return nil, err
	}
	res, err := m.virtuoso.Query(q)
	if err != nil {
		return nil, err
	}
	log.Printf("Fant %d kandidater", len(res.Solutions()))
	var diffs []workDiff
	for _, solution := range res.Solutions() {
		prodWork := solution["prodWork"].String()

		q, err := m.queries.Prepare("prodWork", struct{ URI string }{prodWork})
		if err != nil {
			return nil, err
		}
		fromWork, err := m.virtuoso.Construct(q)
		if err != nil {
			return nil, err
		}

		q, err = m.queries.Prepare("migWork", struct{ URI string }{prodWork})
		if err != nil {
			return nil, err
		}
		toWork, err := m.virtuoso.Construct(q)
		if err != nil {
			return nil, err
		}

		if reflect.DeepEqual(fromWork, toWork) {
			continue
		}

		diff := diffWorks(fromWork, toWork)
		diffs = append(diffs, diff)
	}
	return diffs, nil
}

type prop struct {
	a, b []rdf.Term
}
type workDiff struct {
	Title  string
	ID     string
	From   map[string]string
	To     map[string]string
	diff   map[rdf.IRI]prop
	SPARQL string
}

type Title struct {
	mainTitle  string
	partTitle  string
	subtitle   string
	partNumber string
}

func (t Title) String() string {
	s := t.mainTitle
	if t.subtitle != "" {
		s += " : " + t.subtitle
	}
	if t.partNumber != "" {
		s += ". " + t.partNumber
	}
	if t.partTitle != "" {
		s += ". " + t.partTitle
	}
	return s
}

func skipProp(prop rdf.Term) bool {
	return rdf.TermsEqual(prop, mustURI("genreLabel")) ||
		rdf.TermsEqual(prop, mustURI("subjectLabel")) ||
		rdf.TermsEqual(prop, mustURI("recordId")) ||
		rdf.TermsEqual(prop, mustURI("classificationLabel")) ||
		rdf.TermsEqual(prop, mustURI("classNumberAndSource")) ||
		rdf.TermsEqual(prop, mustURI("comptype"))
}

func diffWorks(from, to []rdf.Triple) workDiff {
	work := workDiff{
		diff: make(map[rdf.IRI]prop),
		From: make(map[string]string),
		To:   make(map[string]string),
	}
	title := Title{}
	for _, t := range from {
		if rdf.TermsEqual(t.Pred, mustURI("mainTitle")) {
			work.ID = t.Subj.String()
			title.mainTitle = t.Obj.String()
			continue
		}
		if rdf.TermsEqual(t.Pred, mustURI("partTitle")) {
			title.partTitle = t.Obj.String()
			continue
		}
		if rdf.TermsEqual(t.Pred, mustURI("subtitle")) {
			title.subtitle = t.Obj.String()
			continue
		}
		if rdf.TermsEqual(t.Pred, mustURI("partNumber")) {
			title.partNumber = t.Obj.String()
			continue
		}
		prop := work.diff[t.Pred.(rdf.IRI)]
		prop.a = append(prop.a, t.Obj)
		work.diff[t.Pred.(rdf.IRI)] = prop
	}

	for _, t := range to {
		prop := work.diff[t.Pred.(rdf.IRI)]
		prop.b = append(prop.b, t.Obj)
		work.diff[t.Pred.(rdf.IRI)] = prop
	}
	work.Title = title.String()
	for k, v := range work.diff {
		sort.Slice(v.a, func(i, j int) bool {
			return v.a[i].String() < v.a[j].String()
		})
		sort.Slice(v.b, func(i, j int) bool {
			return v.b[i].String() < v.b[j].String()
		})
		if reflect.DeepEqual(v.a, v.b) {
			delete(work.diff, k)
		}
	}
	uri := from[0].Subj.Serialize(rdf.NTriples)
	var b bytes.Buffer
	b.WriteString("# Updating work ")
	b.WriteString(uri)
	b.WriteString("\n\nWITH <https://katalog.deichman.no>")
	b.WriteString("\n\nDELETE { ")
	b.WriteString(uri)
	b.WriteString(" ?p ?o . ?class ?cp ?co }\nWHERE { \n{ ")
	b.WriteString(uri)
	b.WriteString(" ?p ?o .\n\tVALUES ?p { <http://migration.deichman.no/splitFrom> <http://data.deichman.no/ontology#hasClassification> ")
	for k, prop := range work.diff {
		for _, term := range prop.a {
			if _, ok := labels[k]; !ok {
				continue
			}
			from := work.From[labels[k]]
			if from != "" {
				from += ", "
			}
			from += term.String()
			work.From[labels[k]] = from
		}
		if skipProp(k) {
			continue
		}
		b.WriteString(k.Serialize(rdf.NTriples))
		b.WriteRune(' ')
	}
	b.WriteString("} }\nUNION { ")
	b.WriteString(uri)
	b.WriteString(" <http://data.deichman.no/ontology#hasClassification> ?class . ?class ?cp ?co .}\n};\n\n")
	b.WriteString("INSERT DATA {")
	for k, v := range work.diff {
		onlyInsert := false
		if _, ok := labels[k]; !ok {
			onlyInsert = true
		}
		for _, term := range v.b {
			if !onlyInsert {
				to := work.To[labels[k]]
				if to != "" {
					to += ", "
				}
				to += term.String()
				work.To[labels[k]] = to
				if skipProp(k) {
					continue
				}
			}
			if rdf.TermsEqual(k, mustURI("classNumberAndSource")) {
				parts := strings.Split(term.String(), "____")
				b.WriteString("\n\t")
				b.WriteString(uri)
				b.WriteString(" <http://data.deichman.no/ontology#hasClassification> [ a <http://data.deichman.no/ontology#ClassificationEntry> ; <http://data.deichman.no/ontology#hasClassificationNumber> ")
				b.WriteString(strconv.Quote(parts[0]))
				if len(parts) > 1 {
					b.WriteString(" ; <http://data.deichman.no/ontology#hasClassificationSource> <")
					b.WriteString(parts[1])
					b.WriteString(">")
				}
				b.WriteString(" ]")
				b.WriteString(" .")
				continue
			}
			b.WriteString("\n\t")
			b.WriteString(uri)
			b.WriteRune(' ')
			b.WriteString(k.Serialize(rdf.NTriples))
			b.WriteRune(' ')
			b.WriteString(term.Serialize(rdf.NTriples))
			b.WriteRune(' ')
			b.WriteString(".")
		}
	}
	b.WriteString("\n};")

	work.SPARQL = b.String()
	return work
}

func (m *Main) handler(w http.ResponseWriter, r *http.Request) {
	diffs, err := m.run()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Title < diffs[j].Title
	})

	if err := templ.Execute(w, diffs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (m *Main) updateHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	queries := r.FormValue("queries")
	works := r.FormValue("works")

	resp, err := http.PostForm("http://virtuoso:8890/sparql",
		url.Values{"update": {queries}})
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		http.Error(w, string(b), http.StatusInternalServerError)
		return
	}

	workURIs := strings.Split(works, "\n")
	for _, work := range workURIs {
		id := strings.TrimPrefix(work, "http://data.deichman.no/")
		req, err := http.NewRequest(http.MethodPatch, "http://services:8005/"+strings.TrimSuffix(id, "\r"), bytes.NewBuffer([]byte("[]")))
		req.Header.Set("Content-Type", "application/ldpatch+json")
		if err != nil {
			log.Println(err)
			continue
		}
		_, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			continue
		}
	}

	fmt.Fprintf(w, "OK oppdaterte alle verk")
}

func main() {
	log.SetFlags(0)

	m, err := newMain()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", m.handler)
	http.HandleFunc("/update", m.updateHandler)

	log.Fatal(http.ListenAndServe(":8811", nil))

}
