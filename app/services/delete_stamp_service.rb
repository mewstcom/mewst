# typed: strict
# frozen_string_literal: true

class DeleteStampService < ApplicationService
  class Input < T::Struct
    extend T::Sig

    const :profile, Profile
    const :target_post, Post

    sig { params(form: Latest::StampForm).returns(Input) }
    def self.from_latest_form(form:)
      new(
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
    stamp = input.profile.stamps.where(post: input.target_post).sole
    stamp.destroy!

    Result.new(post: input.target_post.reload)
  end
end
