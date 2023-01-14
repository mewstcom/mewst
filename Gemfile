# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.2.0"

gem "rails", "~> 7.0.0"

gem "activerecord-session_store"
gem "bootsnap", require: false
gem "connection_pool" # Used by Redis
gem "cssbundling-rails"
gem "email_validator"
gem "enumerize"
gem "http_accept_language"
gem "image_processing" # Used by Shrine
gem "jb"
gem "jsbundling-rails"
gem "pg"
gem "propshaft"
gem "puma"
gem "pundit"
gem "rails-i18n"
gem "redis"
gem "shrine"
gem "sidekiq"
gem "sorbet-runtime"
gem "twilio-ruby"
gem "twitter_oauth2"
gem "view_component"

group :development, :test do
  gem "dotenv-rails"
  gem "factory_bot_rails"
  gem "rspec-rails"
  gem "rubocop-rails", require: false
  gem "rubocop-rspec", require: false
  gem "rubocop-sorbet", require: false
  gem "standard"
end

group :development do
  gem "bullet"
  gem "sorbet"
  gem "tapioca", "~> 0.10.0", require: false
  gem "web-console"
end
