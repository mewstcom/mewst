# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :profile_record, class: "ProfileRecord" do
    sequence(:atname) { |n| "test_#{n}" }
    owner_type { ProfileOwnerType::User.serialize }
    joined_at { Time.current }
  end
end
