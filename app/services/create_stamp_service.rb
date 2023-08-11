# typed: strict
# frozen_string_literal: true

class CreateStampService < ApplicationService
  class Input < T::Struct
    const :original_post, Post
    const :profile, Profile
    const :target_post, Post

    sig { params(form: Latest::StampForm).returns(Input) }
    def self.from_latest_form(form:)
      new(
        original_post: form.original_post,
        viewer: form.profile,
        follow: form.follow
      )
    end
  end

  class Result < T::Struct
    const :post, Post
  end

  sig { params(input: Input).returns(Result) }
  def call
    comment_post = input.original_post.comment_post!
    input.profile.stamps.where(comment_post:).first_or_create!(stamped_at: Time.current)

    Result.new(post: input.target_post)
  end
end
