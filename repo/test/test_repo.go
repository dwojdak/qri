package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ipfs/go-datastore"
	"github.com/qri-io/analytics"
	"github.com/qri-io/cafs/memfs"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func NewTestRepo() (mr repo.Repo, err error) {
	datasets := []string{"movies", "cities", "counter"}
	p := &profile.Profile{
		Username: "test_user",
	}
	ms := memfs.NewMapstore()
	mr, err = repo.NewMemRepo(p, ms, repo.MemPeers{}, &analytics.Memstore{})
	if err != nil {
		return
	}

	var (
		rawdata, dsdata []byte
		datakey, dskey  datastore.Key
	)
	for _, k := range datasets {
		rawdata, err = ioutil.ReadFile(pkgPath(fmt.Sprintf("testdata/%s.csv", k)))
		if err != nil {
			return
		}

		datakey, err = ms.Put(memfs.NewMemfileBytes(k, rawdata), true)
		if err != nil {
			return
		}

		dsdata, err = ioutil.ReadFile(pkgPath(fmt.Sprintf("testdata/%s.json", k)))
		if err != nil {
			return
		}

		ds := &dataset.Dataset{}
		if err = json.Unmarshal(dsdata, ds); err != nil {
			return
		}
		ds.Data = datakey

		dskey, err = dsfs.SaveDataset(ms, ds, true)
		if err != nil {
			return
		}
		if err = mr.PutName(k, dskey); err != nil {
			return
		}
	}

	return
}

func pkgPath(paths ...string) string {
	gp := os.Getenv("GOPATH")
	return filepath.Join(append([]string{gp, "src/github.com/qri-io/qri/repo/test"}, paths...)...)
}

var BadDataFile = memfs.NewMemfileBytes("bad_csv_file.csv", []byte(`
asdlkfasd,,
fm as
f;lajsmf 
a
's;f a'
sdlfj asdf`))

var BadDataFormatFile = memfs.NewMemfileBytes("abc.csv", []byte(`
"colA","colB","colC","colD"
1,2,3,4
1,2,3`))

var BadStructureFile = memfs.NewMemfileBytes("badStructure.csv", []byte(`
colA, colB, colB, colC
1,2,3,4
1,2,3,4`))

var JobsByAutomationFile = memfs.NewMemfileBytes("jobs_ranked_by_automation_probability.csv", []byte(`rank,probability_of_automation,soc_code,job_title
702,"0.99","41-9041","Telemarketers"
701,"0.99","23-2093","Title Examiners, Abstractors, and Searchers"
700,"0.99","51-6051","Sewers, Hand"
699,"0.99","15-2091","Mathematical Technicians"
698,"0.99","13-2053","Insurance Underwriters"
697,"0.99","49-9064","Watch Repairers"
696,"0.99","43-5011","Cargo and Freight Agents"
695,"0.99","13-2082","Tax Preparers"
694,"0.99","51-9151","Photographic Process Workers and Processing Machine Operators"
693,"0.99","43-4141","New Accounts Clerks"
692,"0.99","25-4031","Library Technicians"
691,"0.99","43-9021","Data Entry Keyers"
690,"0.98","51-2093","Timing Device Assemblers and Adjusters"
689,"0.98","43-9041","Insurance Claims and Policy Processing Clerks"
688,"0.98","43-4011","Brokerage Clerks"
687,"0.98","43-4151","Order Clerks"
686,"0.98","13-2072","Loan Officers"
685,"0.98","13-1032","Insurance Appraisers, Auto Damage"
684,"0.98","27-2023","Umpires, Referees, and Other Sports Officials"
683,"0.98","43-3071","Tellers"
682,"0.98","51-9194","Etchers and Engravers"
681,"0.98","51-9111","Packaging and Filling Machine Operators and Tenders"
680,"0.98","43-3061","Procurement Clerks"
679,"0.98","43-5071","Shipping, Receiving, and Traffic Clerks"
678,"0.98","51-4035","Milling and Planing Machine Setters, Operators, and Tenders, Metal and Plastic"
677,"0.98","13-2041","Credit Analysts"
676,"0.98","41-2022","Parts Salespersons"
675,"0.98","13-1031","Claims Adjusters, Examiners, and Investigators"
674,"0.98","53-3031","Driver/Sales Workers"
673,"0.98","27-4013","Radio Operators"
`))

var JobsByAutomationFileLower = memfs.NewMemfileBytes("jobs_ranked_by_automation_probabilityVersion2.csv", []byte(`rank,probability_of_automation,soc_code,job_title
702,"0.99","41-9041","telemarketers"
701,"0.99","23-2093","title examiners, abstractors, and searchers"
700,"0.99","51-6051","sewers, hand"
699,"0.99","15-2091","mathematical technicians"
698,"0.99","13-2053","insurance underwriters"
697,"0.99","49-9064","watch repairers"
696,"0.99","43-5011","cargo and freight agents"
695,"0.99","13-2082","tax preparers"
694,"0.99","51-9151","photographic process workers and processing machine operators"
693,"0.99","43-4141","new accounts clerks"
692,"0.99","25-4031","library technicians"
691,"0.99","43-9021","data entry keyers"
690,"0.98","51-2093","timing device assemblers and adjusters"
689,"0.98","43-9041","insurance claims and policy processing clerks"
688,"0.98","43-4011","brokerage clerks"
687,"0.98","43-4151","order clerks"
686,"0.98","13-2072","loan officers"
685,"0.98","13-1032","insurance appraisers, auto damage"
684,"0.98","27-2023","umpires, referees, and other sports officials"
683,"0.98","43-3071","tellers"
682,"0.98","51-9194","etchers and engravers"
681,"0.98","51-9111","packaging and filling machine operators and tenders"
680,"0.98","43-3061","procurement clerks"
679,"0.98","43-5071","shipping, receiving, and traffic clerks"
678,"0.98","51-4035","milling and planing machine setters, operators, and tenders, metal and plastic"
677,"0.98","13-2041","credit analysts"
676,"0.98","41-2022","parts salespersons"
675,"0.98","13-1031","claims adjusters, examiners, and investigators"
674,"0.98","53-3031","driver/sales workers"
673,"0.98","27-4013","radio operators"
`))

var JobsByAutomationFileLower2 = memfs.NewMemfileBytes("jobs_ranked_by_automation_probabilityVersion3.csv", []byte(`rank,probability_of_automation,soc_code,job_title
702,"0.99","41-9041","telemarketers"
701,"0.99","23-2093","title examiners, abstractors, and searchers"
700,"0.99","51-6051","sewers, hand"
699,"0.99","15-2091","mathematical technicians"
698,"0.99","13-2053","insurance underwriters"
697,"0.99","49-9064","watch repairers"
696,"0.99","43-5011","cargo and freight agents"
695,"0.99","13-2082","tax preparers"
694,"0.99","51-9151","photographic process workers and processing machine operators"
693,"0.99","43-4141","new accounts clerks"
692,"0.99","25-4031","library technicians"
691,"0.99","43-9021","data entry keyers"
690,"0.98","51-2093","timing device assemblers and adjusters"
689,"0.98","43-9041","insurance claims and policy processing clerks"
688,"0.98","43-4011","brokerage clerks"
687,"0.98","43-4151","order clerks"
686,"0.98","13-2072","loan officers"
685,"0.98","13-1032","insurance appraisers, auto damage"
684,"0.98","27-2023","umpires, referees, and other sports officials"
683,"0.98","43-3071","tellers"
682,"0.98","51-9194","etchers and engravers"
681,"0.98","51-9111","packaging and filling machine operators and tenders"
680,"0.98","43-3061","procurement clerks"
679,"0.98","43-5071","shipping, receiving, and traffic clerks"
678,"0.98","51-4035","milling and planing machine setters, operators, and tenders, metal and plastic"
677,"0.98","13-2041","credit analysts"
676,"0.98","41-2022","parts salespersons"
675,"0.98","13-1031","claims adjusters, examiners, and investigators"
674,"0.98","53-3031","driver/sales workers"
673,"0.98","27-4013","radio operators"
`))
