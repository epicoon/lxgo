package ws

func SendMessage(m IMessage) {
	ids := m.ReceiverIDs()
	for _, id := range ids {
		if !m.ValidateConnectionID(id) {
			continue
		}

		data := m.PrepareDataForConnection(id)
		conn := m.Server().Connections().Get(id)
		conn.Send(data, "text", false)
	}
}
