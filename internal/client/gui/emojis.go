package gui

var emojiMap = map[string]map[string]string{
	"😊": {
		":)": "😊", ":(": "😢", "XD": "😂", ":D": "😄", "<3": "❤️", "</3": "💔",
		":P": "😛", ";)": "😉", ":o": "😲", ":/": "😕", ":|": "😐", ":angry:": "😠", 
		":cry:": "😭", ":sick:": "🤢", ":sleepy:": "😴", ":cool:": "😎", ":confused:": "😕",
	},
	"👍": {
		":like:": "👍", ":dislike:": "👎", ":clap:": "👏", ":pray:": "🙏",
		":ok:": "👌", ":fist:": "✊", ":wave:": "👋", ":muscle:": "💪",
		":point_up:": "☝️", ":point_down:": "👇", ":point_left:": "👈", ":point_right:": "👉",
	},
	"🐱": {
		":dog:": "🐶", ":cat:": "🐱", ":fox:": "🦊", ":lion:": "🦁", ":tiger:": "🐯",
		":bear:": "🐻", ":panda:": "🐼", ":unicorn:": "🦄", ":rabbit:": "🐰",
		":frog:": "🐸", ":monkey:": "🐵", ":elephant:": "🐘", ":koala:": "🐨",
		":penguin:": "🐧", ":whale:": "🐳", ":dolphin:": "🐬", ":octopus:": "🐙",
		":fish:": "🐟", ":butterfly:": "🦋", ":snail:": "🐌", ":bee:": "🐝", ":ant:": "🐜",
	},
	"🍕": {
		":apple:": "🍎", ":banana:": "🍌", ":cherry:": "🍒", ":coffee:": "☕",
		":pizza:": "🍕", ":burger:": "🍔", ":fries:": "🍟", ":cake:": "🍰",
		":icecream:": "🍨", ":donut:": "🍩", ":cookie:": "🍪", ":beer:": "🍺", ":wine:": "🍷",
	},
	"☀️": {
		":sun:": "☀️", ":moon:": "🌙", ":star:": "⭐", ":fire:": "🔥",
		":rainbow:": "🌈", ":gift:": "🎁", ":music:": "🎵", ":game:": "🎮",
	},
	"🏀": {
		":soccer:": "⚽", ":basketball:": "🏀", ":football:": "🏈", ":tennis:": "🎾",
		":baseball:": "⚾", ":golf:": "⛳", ":volleyball:": "🏐", ":pingpong:": "🏓",
		":hockey:": "🏒", ":ski:": "🎿", ":swim:": "🏊", ":bike:": "🚴", ":weight:": "🏋️",
		":martial_arts:": "🥋", ":fencing:": "🤺", ":bowling:": "🎳", ":cricket:": "🏏",
	},
}