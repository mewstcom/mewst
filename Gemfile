# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.2.2"

gem "rails", "~> 7.1.2"

gem "activerecord-session_store"
gem "addressable"
gem "alba"
gem "bcrypt" # `has_secure_password` で使っている
gem "bootsnap", require: false
gem "connection_pool" # Redisで使っている
gem "counter_culture"
gem "cssbundling-rails"
gem "discard"
gem "doorkeeper"
gem "email_validator"
gem "enumerize"
gem "faraday"
gem "good_job"
gem "http_accept_language"
gem "jb"
gem "meta-tags"
gem "oj" # `alba` で使っている
gem "pg"
gem "propshaft"
gem "puma"
gem "pundit"
gem "rack-rewrite"
gem "rails-i18n"
gem "rails_autolink"
gem "redis"
gem "sentry-rails"
gem "sentry-ruby"
gem "stimulus-rails"
gem "strong_migrations"
gem "turbo-rails"
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
  gem "tapioca", ">= 0.10.5", require: false
  gem "web-console"
end

group :production do
  gem "lograge"
  gem "resend"
end

# ARM64な環境でSorbetを使うためのワークアラウンド
# https://github.com/sorbet/sorbet/issues/4119
source "https://gem.fury.io/sorbet-multiarch/" do
  gem "sorbet-runtime"
  gem "sorbet", group: :development
end
