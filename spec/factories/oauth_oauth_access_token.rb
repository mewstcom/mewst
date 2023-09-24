# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :oauth_access_token do
    resource_owner(factory: :actor)
    application(factory: :oauth_application)
    sequence(:token) { |n| "token_#{n}" }
  end
end
