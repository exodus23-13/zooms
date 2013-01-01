# Zooms

Zooms preloads your Rails app so that your normal development tasks such as `console`, `server`, `generate`, and specs/tests take **less than one second**.

This screencast gives a quick overview of how to use zooms with Rails.

[![Watch the screencast!](http://s3.amazonaws.com/burkelibbey/vimeo-zooms.png)](http://vimeo.com/burkelibbey/zooms)

More technically speaking, Zooms is a language-agnostic application checkpointer for non-multithreaded applications. Currently only ruby is targeted, but explicit support for other languages is on the horizon.

## Requirements (for use with Rails)

* OS X 10.7+ *OR* Linux 2.6.13+
* Rails 3.x
* Ruby 1.9.3+ with backported GC from Ruby 2.0 *OR* Rubinius

You can install the GC-patched ruby from [this gist for rbenv](https://gist.github.com/1688857) or [this gist for RVM](https://gist.github.com/4136373). This is not actually 100% necessary, especially if you have a lot of memory. Feel free to give it a shot first without, but if you're suddenly out of RAM, switching to the GC-patched ruby will fix it.

## Installation

Install the gem.

    gem install zooms

Q: "I should put it in my `Gemfile`, right?"

A: No. You can, but running `bundle exec zooms` instead of `zooms` adds precious seconds to commands that otherwise would be quite a bit faster. Zooms was built to be run from outside of bundler.

## Usage

Start the server:

    zooms start

The server will print a list of available commands.

Run some commands in another shell:

    zooms console
    zooms server
    zooms test test/unit/widget_test.rb
    zooms test spec/widget_spec.rb
    zooms generate model omg
    zooms rake -T
    zooms runner omg.rb

## Hacking

To add/modify commands, see [`docs/ruby/modifying.md`](/burke/zooms/tree/master/docs/ruby/modifying.md).

To get started hacking on Zooms itself, see [`docs/overview.md`](/burke/zooms/tree/master/docs/overview.md).

See also the handy contribution guide at [`contributing.md`](/burke/zooms/tree/master/contributing.md).

## Alternative plans

The default plan bundled with zooms only supports Rails 3.x. There is a project (currently WIP) to provide Rails 2.3 support at https://github.com/tyler-smith/zooms-rails23.

---

[![endorse](http://api.coderwall.com/burke/endorsecount.png)](http://coderwall.com/burke)
