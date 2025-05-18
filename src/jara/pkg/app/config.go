package app

type Config struct {
	Theme  Theme
	KeyMap KeyMap
}

var DefaultConfig = Config{
	Theme:  CanonicalTheme,
	KeyMap: DefaultKeyMap,
}
