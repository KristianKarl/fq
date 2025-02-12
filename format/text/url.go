package text

import (
	"net/url"

	"github.com/wader/fq/internal/gojqextra"
	"github.com/wader/fq/pkg/interp"
)

func init() {
	interp.RegisterFunc0("fromurlencode", func(_ *interp.Interp, c string) any {
		u, err := url.QueryUnescape(c)
		if err != nil {
			return err
		}
		return u
	})
	interp.RegisterFunc0("tourlencode", func(_ *interp.Interp, c string) any {
		return url.QueryEscape(c)
	})

	interp.RegisterFunc0("fromurlpath", func(_ *interp.Interp, c string) any {
		u, err := url.PathUnescape(c)
		if err != nil {
			return err
		}
		return u
	})
	interp.RegisterFunc0("tourlpath", func(_ *interp.Interp, c string) any {
		return url.PathEscape(c)
	})

	fromURLValues := func(q url.Values) any {
		qm := map[string]any{}
		for k, v := range q {
			if len(v) > 1 {
				vm := []any{}
				for _, v := range v {
					vm = append(vm, v)
				}
				qm[k] = vm
			} else {
				qm[k] = v[0]
			}
		}

		return qm
	}
	interp.RegisterFunc0("fromurlquery", func(_ *interp.Interp, c string) any {
		q, err := url.ParseQuery(c)
		if err != nil {
			return err
		}
		return fromURLValues(q)
	})
	toURLValues := func(c map[string]any) url.Values {
		qv := url.Values{}
		for k, v := range c {
			if va, ok := gojqextra.Cast[[]any](v); ok {
				var ss []string
				for _, s := range va {
					if s, ok := gojqextra.Cast[string](s); ok {
						ss = append(ss, s)
					}
				}
				qv[k] = ss
			} else if vs, ok := gojqextra.Cast[string](v); ok {
				qv[k] = []string{vs}
			}
		}
		return qv
	}
	interp.RegisterFunc0("tourlquery", func(_ *interp.Interp, c map[string]any) any {
		// TODO: nicer
		c, ok := gojqextra.NormalizeToStrings(c).(map[string]any)
		if !ok {
			panic("not map")
		}
		return toURLValues(c).Encode()
	})

	interp.RegisterFunc0("fromurl", func(_ *interp.Interp, c string) any {
		u, err := url.Parse(c)
		if err != nil {
			return err
		}

		m := map[string]any{}
		if u.Scheme != "" {
			m["scheme"] = u.Scheme
		}
		if u.User != nil {
			um := map[string]any{
				"username": u.User.Username(),
			}
			if p, ok := u.User.Password(); ok {
				um["password"] = p
			}
			m["user"] = um
		}
		if u.Host != "" {
			m["host"] = u.Host
		}
		if u.Path != "" {
			m["path"] = u.Path
		}
		if u.RawPath != "" {
			m["rawpath"] = u.RawPath
		}
		if u.RawQuery != "" {
			m["rawquery"] = u.RawQuery
			m["query"] = fromURLValues(u.Query())
		}
		if u.Fragment != "" {
			m["fragment"] = u.Fragment
		}
		return m
	})
	interp.RegisterFunc0("tourl", func(_ *interp.Interp, c map[string]any) any {
		// TODO: nicer
		c, ok := gojqextra.NormalizeToStrings(c).(map[string]any)
		if !ok {
			panic("not map")
		}

		str := func(v any) string { s, _ := gojqextra.Cast[string](v); return s }
		u := url.URL{
			Scheme:   str(c["scheme"]),
			Host:     str(c["host"]),
			Path:     str(c["path"]),
			Fragment: str(c["fragment"]),
		}

		if um, ok := gojqextra.Cast[map[string]any](c["user"]); ok {
			username, password := str(um["username"]), str(um["password"])
			if username != "" {
				if password == "" {
					u.User = url.User(username)
				} else {
					u.User = url.UserPassword(username, password)
				}
			}
		}
		if s, ok := gojqextra.Cast[string](c["rawquery"]); ok {
			u.RawQuery = s
		}
		if qm, ok := gojqextra.Cast[map[string]any](c["query"]); ok {
			u.RawQuery = toURLValues(qm).Encode()
		}

		return u.String()
	})
}
