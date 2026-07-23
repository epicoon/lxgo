package ws

/** @constructor */
func NewChannelBuilder() IChannelBuilder {
	return &channelBuilder{}
}

/** @interface IChannelBuilder */
type channelBuilder struct {
	creator     IConnection
	key         string
	public      bool
	proprietary bool
	sharedData  map[string]any
	initData    map[string]any
}

var _ IChannelBuilder = (*channelBuilder)(nil)

func (b *channelBuilder) Creator() IConnection {
	return b.creator
}

func (b *channelBuilder) SetCreator(c IConnection) IChannelBuilder {
	b.creator = c
	return b
}

func (b *channelBuilder) Key() string {
	return b.key
}

func (b *channelBuilder) SetKey(key string) IChannelBuilder {
	b.key = key
	return b
}

func (b *channelBuilder) Public() bool {
	return b.public
}

func (b *channelBuilder) SetPublic(pub bool) IChannelBuilder {
	b.public = pub
	return b
}

func (b *channelBuilder) Proprietary() bool {
	return b.proprietary
}

func (b *channelBuilder) SetProprietary(prop bool) IChannelBuilder {
	b.proprietary = prop
	return b
}

func (b *channelBuilder) SharedData() map[string]any {
	return b.sharedData
}

func (b *channelBuilder) SetSharedData(data map[string]any) IChannelBuilder {
	b.sharedData = data
	return b
}

func (b *channelBuilder) InitData() map[string]any {
	return b.initData
}

func (b *channelBuilder) SetInitData(data map[string]any) IChannelBuilder {
	b.initData = data
	return b
}
