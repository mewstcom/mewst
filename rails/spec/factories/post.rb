# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :post_record, class: "PostRecord" do
    profile_record
    sequence(:content) { |n| "test_#{n}" }
    published_at { Time.current }
    oauth_application
  end
end
