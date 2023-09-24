# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :actor do
    user { association :user }
    profile { user.profile }

    trait :with_access_token_for_web do
      after(:create) do |actor|
        application = create(:oauth_application, :mewst_web)
        create(:oauth_access_token, application:, resource_owner: actor)
      end
    end
  end
end
