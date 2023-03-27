# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :verification do
    sequence(:email) { |n| "test-#{n}@example.com" }
    event { :sign_up }
    sequence(:code) { |n| "00000#{n}" }
  end
end
