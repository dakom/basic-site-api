## Synopsis

Basic starter code for standard api site on appengine. Includes account essentials out of the box (forgot password, mailing list subscription, avatar, etc.)

Currently it's a work-in-progress and hasn't been used in very different kinds of scenarios, which would likely affect some architectural changes, but it is being used on some live projects at the moment.

## Code Example

An entire api site can be run by something like this in your appengine project's main.go:

```
package main

import (
	basic_setup "github.com/dakom/basic-site-api/setup"
)

func init() {
	basic_setup.Start(MY_PAGE_CONFIGS, MY_SITE_CONFIG)
}
```

Where `MY_PAGE_CONFIGS` is a `map[string]*pages.PageConfig` and `MY_SITE_CONFIG` is a properly setup `*custom.Config`

You'd want to extend the supplied `MY_PAGE_CONFIGS` to handle all the requests your site deals with that aren't part of the base.

## Motivation

The idea is to create a framework for handling most of the common scenarios, and centralize key features (like authorization, jwt refreshing, different http responses, etc.) - not just as boilerplate but as a package which can be imported and used.

The current structure is mostly around getting requests and giving back json - though there are some areas where other responses are used (e.g. communicating with third parties), and one could use it as a starting point for a completely different kind of site (e.g. one which uses html templates)


## TODO

* Write docs
* Give examples
* Improve code

## Authors

David Komer <david.komer@gmail.com>

## License

MIT License
