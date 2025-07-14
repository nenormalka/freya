package metadata

type (
	Platform interface {
		IsValid() bool
		IsMobile() bool
		IsIos() bool
		IsAndroid() bool
		IsWeb() bool
		IsMobileWeb() bool
		IsDesktopWeb() bool
		Platform() string
	}
)

func (p PlatformType) Platform() string {
	return string(p)
}

func (p PlatformType) IsValid() bool {
	return p.IsMobile() || p.IsWeb()
}

func (p PlatformType) IsMobile() bool {
	return p.IsIos() || p.IsAndroid()
}

func (p PlatformType) IsWeb() bool {
	return p.IsMobileWeb() || p.IsDesktopWeb()
}

func (p PlatformType) IsIos() bool {
	return p == PlatformTypeIOS
}

func (p PlatformType) IsAndroid() bool {
	return p == PlatformTypeANDROID
}

func (p PlatformType) IsMobileWeb() bool {
	return p == PlatformTypeWEBMobile || p == PlatformTypeWEBMobile2
}

func (p PlatformType) IsDesktopWeb() bool {
	switch p {
	case PlatformTypeWEB, PlatformTypeDesktop, PlatformTypeWebDesktop, PlatformTypeWebDesktop2:
		return true
	default:
		return false
	}
}
