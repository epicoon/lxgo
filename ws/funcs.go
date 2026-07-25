package ws

// SendMessage delivers m to every connection ReceiverIDs() names (skipping
// any ValidateConnectionID rejects), each getting its own
// PrepareDataForConnection result.
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
