# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :phone_number_confirmation do
    sequence(:phone_number) { |n| "+81900000000#{n}" }
    sequence(:verification_code) { |n| "00000#{n}" }
  end
end
