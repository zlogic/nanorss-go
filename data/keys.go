package data

const separator = ":"

func CreateKey(v interface{}, id string) []byte {
	switch v.(type) {
	case *User:
		return []byte("user" + separator + id)
	default:
		return nil
	}
}
