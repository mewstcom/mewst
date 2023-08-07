# typed: strict
# frozen_string_literal: true

class Latest::Presenters::Repost < Latest::Presenters::Base
  sig { params(post: Post).void }
  def initialize(post:)
    @post = post
  end

  sig { params(args: T.untyped).returns(T::Hash[Symbol, T.untyped]) }
  def as_json(*args)
    {
      post: Resources::Latest::Post.new(post).to_h
    }
  end

  sig { returns(Post) }
  attr_reader :post
  private :post
end
