# typed: strict
# frozen_string_literal: true

class AddPostToHomeService < ApplicationService
  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error], default: []
  end

  sig { params(form: PostToHomeForm).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    post = T.must(@form.post)
    profile = T.must(@form.profile)

    T.cast(Mewst::Redis.client, Redis).zadd(profile.inbox_key, post.inbox_item_score, post.id)

    Result.new
  end
end
