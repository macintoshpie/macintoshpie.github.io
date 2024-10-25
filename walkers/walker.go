package walkers

import parser "github.com/macintoshpie/listwebsite/parsers"

type ListenerCallback func(node *parser.Node)

type ListenerConfig struct {
	OnEnter ListenerCallback
	OnExit  ListenerCallback
}

type Walker struct {
	listenerConfigs map[parser.BlockType][]ListenerConfig
}

func NewWalker() *Walker {
	return &Walker{
		listenerConfigs: make(map[parser.BlockType][]ListenerConfig),
	}
}

func (lr *Walker) AddEventListener(blockTypes []parser.BlockType, listener ListenerConfig) {
	for _, blockType := range blockTypes {
		lr.listenerConfigs[blockType] = append(lr.listenerConfigs[blockType], listener)
	}
}

func (lr *Walker) Walk(node *parser.Node) {
	listenerConfigs, ok := lr.listenerConfigs[node.BlockType]

	if ok {
		for _, listenerConfig := range listenerConfigs {
			if listenerConfig.OnEnter == nil {
				continue
			}

			listenerConfig.OnEnter(node)
		}
	}

	for _, child := range node.Children {
		lr.Walk(child)
	}

	if ok {
		for _, listenerConfig := range listenerConfigs {
			if listenerConfig.OnExit == nil {
				continue
			}

			listenerConfig.OnExit(node)
		}
	}
}
