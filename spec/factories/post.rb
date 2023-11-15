# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :post do
    profile
    sequence(:content) { |n| "test_#{n}" }
    published_at { Time.current }
    oauth_application
  end
end
