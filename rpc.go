package tetris

const CowsayName = "cowsay"

type CowsayRequest struct {
	Key string
}
type CowsayResponse struct {
	Say string
}
