# typed: strict
# frozen_string_literal: true

class CreateLinkUseCase < ApplicationUseCase
  class Result < T::Struct
    const :link, LinkRecord
  end

  sig { params(canonical_url: String, domain: String, title: String, image_url: String).returns(Result) }
  def call(canonical_url:, domain:, title:, image_url:)
    link = LinkRecord.new(canonical_url:, domain:, title:, image_url:)

    link.save!

    Result.new(link:)
  end
end
