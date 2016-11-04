package main

type passwordValue string

func newPasswordValue(val string, p *string) *passwordValue {
	*p = val
	return (*passwordValue)(p)
}

func (i *passwordValue) String() string {
	if *i == "" {
		return ""
	} else {
		return "***"
	}
}

func (i *passwordValue) Set(s string) error {
	*i = passwordValue(s)
	return nil
}

func (i *passwordValue) Type() string {
	return "password"
}
