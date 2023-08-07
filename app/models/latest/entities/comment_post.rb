# typed: strict
# frozen_string_literal: true

class Latest::Entities::CommentPost < Latest::Entities::Base
  delegate :comment, to: :comment_post

  sig { params(comment_post: CommentPost).void }
  def initialize(comment_post:)
    @comment_post = comment_post
  end

  sig { returns(CommentPost) }
  attr_reader :comment_post
  private :comment_post
end
