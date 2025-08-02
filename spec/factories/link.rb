# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :link_record, class: "LinkRecord" do
    sequence(:canonical_url) { |n| "https://example.com/#{n}" }
    domain { "example.com" }
    sequence(:title) { |n| "Page Title ##{n}" }
    image_url { "https://example.com/image.jpg" }
  end
end
