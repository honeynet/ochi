package entities

import "math/rand/v2"

var nameAdjectives = []string{
	"admiring", "adoring", "agitated", "amazing", "angry", "awesome", "blissful", "bold",
	"boring", "brave", "busy", "charming", "clever", "cool", "compassionate", "competent",
	"condescending", "confident", "crazy", "dazzling", "determined", "distracted", "dreamy",
	"eager", "ecstatic", "elastic", "elated", "elegant", "eloquent", "epic", "exciting",
	"fervent", "festive", "flamboyant", "focused", "friendly", "frosty", "funny", "gallant",
	"gifted", "goofy", "gracious", "happy", "hardcore", "heuristic", "hopeful", "hungry",
	"infallible", "inspiring", "intelligent", "jolly", "jovial", "keen", "kind", "laughing",
	"loving", "magical", "modest", "musing", "mystifying", "naughty", "nervous", "nifty",
	"nostalgic", "objective", "optimistic", "peaceful", "pedantic", "pensive", "practical",
	"priceless", "quirky", "quizzical", "relaxed", "reverent", "romantic", "serene", "sharp",
	"silly", "sleepy", "stoic", "strange", "stupefied", "suspicious", "sweet", "tender",
	"thirsty", "trusting", "unruffled", "upbeat", "vibrant", "vigilant", "vigorous",
	"wizardly", "wonderful", "xenodochial", "youthful", "zealous", "zen",
}

var nameSurnames = []string{
	"albattani", "allen", "archimedes", "ardinghelli", "babbage", "banach", "bardeen",
	"bartik", "bell", "bhabha", "blackburn", "bohr", "booth", "brahmagupta", "brattain",
	"brown", "carson", "chandrasekhar", "clarke", "cohen", "colden", "cori", "cray",
	"curie", "darwin", "davinci", "dijkstra", "dubinsky", "einstein", "elion", "euclid",
	"euler", "faraday", "fermat", "fermi", "feynman", "franklin", "galileo", "gates",
	"goldberg", "goodall", "haibt", "hamilton", "hawking", "heisenberg", "hodgkin",
	"hopper", "hugle", "hypatia", "jackson", "jang", "jennings", "jepsen", "joliot",
	"kalam", "kapitsa", "kare", "keldysh", "keller", "kepler", "khayyam", "knuth",
	"kowalevski", "lalande", "lamarr", "lamport", "leakey", "lederberg", "lehmann",
	"lewin", "lovelace", "lumiere", "mahavira", "mayer", "mccarthy", "mcclintock",
	"meitner", "mendel", "mestorf", "mirzakhani", "moore", "morse", "newton", "nightingale",
	"nobel", "noether", "northcutt", "noyce", "panini", "pare", "pasteur", "payne",
	"perlman", "pike", "poincare", "poitras", "ptolemy", "raman", "ramanujan", "ride",
	"ritchie", "rhodes", "robinson", "roentgen", "rosalind", "saha", "sammet", "shannon",
	"shirley", "shockley", "sinoussi", "snyder", "spence", "stallman", "swanson",
	"swartz", "swirles", "tesla", "thompson", "torvalds", "turing", "varahamihira",
	"visvesvaraya", "volhard", "wescoff", "wiles", "williams", "wilson", "wing", "wozniak",
	"wright", "yalow", "yonath",
}

// RandomName returns a docker-style name such as "elegant_turing".
func RandomName() string {
	return nameAdjectives[rand.IntN(len(nameAdjectives))] + "_" + nameSurnames[rand.IntN(len(nameSurnames))]
}
