package dawg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/matryer/is"
)

var DataDir = os.Getenv("DATA_PATH")
var DefaultConfig = &config.Config{DataPath: DataDir}

func BenchmarkAnagramBlanks(b *testing.B) {
	// ~1.78 ms on 12thgen-monolith

	d, err := LoadDawg(filepath.Join(DefaultConfig.DataPath, "lexica", "dawg", "CSW21.dawg"))
	if err != nil {
		b.Error("loading CSW21 dawg")
		return
	}
	alph := d.GetRuneMapping()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var anags []string
		da := DawgAnagrammer{}
		if err = da.InitForString(d, "RETINA??"); err != nil {
			b.Error(err)
		} else if err = da.Anagram(d, func(word runemapping.MachineWord) error {
			anags = append(anags, word.UserVisible(alph))
			return nil
		}); err != nil {
			b.Error(err)
		}
	}
}

func TestAnagramBlanks(t *testing.T) {
	is := is.New(t)
	d, err := LoadDawg(filepath.Join(DefaultConfig.DataPath, "lexica", "dawg", "CSW21.dawg"))
	is.NoErr(err)
	alph := d.GetRuneMapping()

	var anags []string
	da := DawgAnagrammer{}
	if err = da.InitForString(d, "RETINA??"); err != nil {
		t.Error(err)
	} else if err = da.Anagram(d, func(word runemapping.MachineWord) error {
		anags = append(anags, word.UserVisible(alph))
		return nil
	}); err != nil {
		t.Error(err)
	}
	is.Equal(anags, strings.Split(
		"ACENTRIC ACTIONER AERATING AERATION ALERTING ALTERING ANGRIEST ANGSTIER ANIMATER ANKERITE ANOESTRI ANOINTER ANORETIC ANTERIOR ANTHERID ANTIHERO ANTIMERE ANTIQUER ANTIRAPE ANTISERA ANTIWEAR ANURETIC APERIENT ARENITES ARENITIC ARETTING ARGENTIC AROINTED ARSENITE ARSONITE ARTESIAN ARTINESS ASTRINGE ATABRINE ATEBRINS ATHERINE ATRAZINE ATROPINE ATTAINER AUNTLIER AVERTING BACTERIN BANISTER BARITONE BARNIEST BERATING BRAUNITE CANISTER CARINATE CARNIEST CATERING CENTIARE CERATINS CISTERNA CITRANGE CLARINET CRANIATE CREATINE CREATING CREATINS CREATION CRINATED DAINTIER DATURINE DENTARIA DERATING DERATION DETAINER DETRAINS DICENTRA DIPTERAN EARTHING ELATERIN EMIGRANT ENARGITE ENTAILER ENTRAILS ENTRAINS EXPIRANT FAINTERS FAINTIER FENITARS GANISTER GANTRIES GNATTIER GRADIENT GRANITES GRATINEE GRIEVANT HAIRNETS HAURIENT HEARTING HERNIATE INAURATE INCREATE INDARTED INDURATE INEARTHS INERRANT INERTIAE INERTIAL INERTIAS INFLATER INGATHER INGRATES INORNATE INTEGRAL INTERACT INTERAGE INTERLAP INTERLAY INTERMAT INTERNAL INTERVAL INTERWAR INTRANET INTREATS ITERANCE JAUNTIER KERATINS KNITWEAR KREATINE LARNIEST LATRINES MARINATE MARTINET MERANTIS MINARETS NAARTJIE NACRITES NARKIEST NARTJIES NAVICERT NITRATED NITRATES NOTAIRES NOTARIES NOTARISE NOTARIZE OBTAINER ORDINATE ORIENTAL PAINTERS PAINTIER PAINTURE PANTRIES PERIANTH PERTAINS PINASTER PRETRAIN PRISTANE QUAINTER RABATINE RAIMENTS RAINDATE RAINIEST RANDIEST RANGIEST RATANIES RATIONED RATLINES RATTLINE REACTING REACTION REANOINT REASTING REATTAIN REBATING REDATING REINSTAL RELATING RELATION REMATING REOBTAIN REPAINTS RESIANTS RESINATA RESINATE RESTRAIN RETAINED RETAINER RETAKING RETAPING RETAXING RETINALS RETINULA RETIRANT RETRAINS RETSINAS ROSINATE RUINATED RUINATES RUMINATE SANTERIA SATINIER SCANTIER SEATRAIN SENORITA SLANTIER SNARIEST STAINERS STARNIES STEARINE STEARING STEARINS STRAINED STRAINER STRAITEN TABERING TABORINE TACRINES TAILERON TAINTURE TANGLIER TAPERING TARTINES TARWHINE TASERING TAURINES TAVERING TENTORIA TENURIAL TERAGLIN TERMINAL TERRAINS TERRAPIN TERTIANS THERIANS TINWARES TRAINEES TRAINERS TRAINMEN TRAMLINE TRANCIER TRANNIES TRANSIRE TRAPLINE TREADING TREATING TREENAIL TRENAILS TRIANGLE TRIAZINE TRIENNIA TRIPLANE TRIPTANE TWANGIER TYRAMINE URANITES URBANITE URINATED URINATES VAUNTIER VAWNTIER VERATRIN VINTAGER WARIMENT WATERING XERANTIC", " "))
	is.Equal(len(anags), 259)
}

type testpair struct {
	prefix string
	found  bool
}

var findWordTests = []testpair{
	{"ZYZZ", false},
	{"ZYZZZ", false},
	{"BAREFIT", true},
	{"KWASH", false},
	{"KWASHO", false},
	{"BAREFITS", false},
	{"AASVOGELATION", false},
	{"FIREFANGNESS", false},
	{"TRIREMED", false},
	{"BAREFITD", false},
	{"KAFF", false},
	{"FF", false},
	{"ABRACADABRA", true},
	{"EEE", false},
	{"ABC", false},
	{"ABCD", false},
	{"FIREFANG", true},
	{"X", false},
	{"Z", false},
	{"Q", false},
	{"KWASHIORKORS", true},
	{"EE", false},
	{"RETIARII", true},
	{"CINEMATOGRAPHER", true},
	{"ANIMADVERTS", true},
	{"PRIVATDOZENT", true},
	{"INEMATOGRAPHER", false},
	{"RIIRAITE", false},
	{"GG", false},
	{"LL", false},
	{"ZZ", false},
	{"ZZZ", true},
	{"ZZZS", false},
}

func TestFindMachineWord(t *testing.T) {
	is := is.New(t)
	d, _ := LoadDawg(filepath.Join(DefaultConfig.DataPath, "lexica", "dawg", "NWL20.dawg"))
	for _, pair := range findWordTests {
		t.Run(pair.prefix, func(t *testing.T) {
			mw, err := runemapping.ToMachineLetters(pair.prefix, d.GetRuneMapping())
			is.NoErr(err)
			found := d.HasWord(mw)
			is.Equal(found, pair.found)
		})
	}
}
