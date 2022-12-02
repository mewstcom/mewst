# typed: strict
# frozen_string_literal: true

class CreatePostService < ApplicationService
  class Error < T::Struct
    const :message, String
  end

  class Result < T::Struct
    const :errors, T::Array[Error], default: []
    const :post, T.nilable(Post)
  end

  sig { params(form: PostForm).void }
  def initialize(form:)
    @form = form
    @profile = T.let(T.must(@form.profile), Profile)
  end

  sig { returns(Result) }
  def call
    post = @profile.posts.create!(body: @form.body)

    Result.new(post:)
  end
end
