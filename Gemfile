# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.2.2"

gem "rails", "~> 7.1.1"

gem "activerecord-session_store"
gem "addressable"
gem "alba"
gem "bcrypt" # `has_secure_password` で使っている
gem "bootsnap", require: false
gem "connection_pool" # Redisで使っている
gem "counter_culture"
gem "discard"
gem "doorkeeper"
gem "email_validator"
gem "enumerize"
gem "faraday"
gem "good_job"
gem "http_accept_language"
gem "oj" # `alba` で使っている
gem "pg"
gem "puma"
gem "pundit"
gem "rails-i18n"
gem "rails_autolink"
gem "redis"
gem "sentry-rails"
gem "sentry-ruby"
gem "sorbet-runtime"
gem "strong_migrations"

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
  gem "sorbet"
  gem "tapioca", ">= 0.10.5", require: false
  gem "web-console"
end

group :production do
  gem "lograge"
  gem "resend"
end
