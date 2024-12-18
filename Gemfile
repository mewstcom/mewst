# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.3.6"

gem "rails", "~> 8.0.1"

gem "activerecord_cursor_paginate"
gem "addressable"
gem "alba"
gem "bcrypt" # `has_secure_password` で使っている
gem "bootsnap", require: false
gem "cssbundling-rails"
gem "discard"
gem "doorkeeper"
gem "email_validator"
gem "enumerize"
gem "faraday"
gem "good_job"
gem "http_accept_language"
gem "inline_svg"
gem "jb"
gem "jsbundling-rails"
gem "meta-tags"
gem "nokogiri"
gem "oj" # `alba` で使っている
gem "pg"
gem "propshaft"
gem "puma"
gem "pundit"
gem "rack-rewrite"
gem "rails-i18n"
gem "rails_autolink"
gem "sentry-rails"
gem "sentry-ruby"
gem "sorbet-runtime"
gem "strong_migrations"
gem "view_component"

group :development, :test do
  gem "committee"
  gem "committee-rails"
  gem "dotenv-rails"
  gem "factory_bot_rails"
  gem "rspec-rails"
  gem "rubocop-factory_bot", require: false
  gem "rubocop-rspec", require: false
  gem "standard"
  gem "standard-rails"
  gem "standard-sorbet"
end

group :development do
  gem "bullet"
  gem "letter_opener_web"
  gem "rails_live_reload"
  gem "sorbet"
  gem "tapioca", require: false
  gem "web-console"
end

group :test do
  gem "cuprite"
  gem "capybara"
  gem "vcr"
end

group :production do
  gem "lograge"
  gem "resend"
end
