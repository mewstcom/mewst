# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.2.2"

gem "rails", "~> 7.0.4"

gem "activerecord-session_store"
gem "bcrypt" # Used by `has_secure_password`
gem "bootsnap", require: false
gem "connection_pool" # Used by Redis
gem "counter_culture"
gem "doorkeeper"
gem "email_validator"
gem "enumerize"
gem "faraday"
gem "google-cloud-pubsub"
gem "google-cloud-tasks"
gem "http_accept_language"
gem "pagy"
gem "panko_serializer"
gem "pg"
gem "puma"
gem "pundit"
gem "rails-i18n"
gem "rails_autolink"
gem "redis"
gem "sorbet-runtime"
gem "sucker_punch"

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
