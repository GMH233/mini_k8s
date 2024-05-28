package pv

type Controller interface {
	Run()
}

type pvController struct {
}

func NewPVController() Controller {
	pc := &pvController{}
	return pc
}

func (pc *pvController) Run() {

}
