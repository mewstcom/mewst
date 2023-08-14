# typed: strict
# frozen_string_literal: true

class CreateStampService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :original_post, Post
    const :profile, Profile
    const :target_post, Post

    sig { params(form: Latest::StampForm).returns(Input) }
    def self.from_latest_form(form:)
      new(
        original_post: form.original_post,
        profile: form.profile.not_nil!,
        target_post: form.target_post.not_nil!
      )
    end
  end

  class Result < T::Struct
    const :post, Post
  end

  sig { params(input: Input).returns(Result) }
  def call(input:)
    comment_post = input.original_post.comment_post.not_nil!
    input.profile.stamps.where(comment_post:).first_or_create!(stamped_at: Time.current)

    Result.new(post: input.target_post)
  end
end
