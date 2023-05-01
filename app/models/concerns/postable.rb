# typed: strict
# frozen_string_literal: true

module Postable
  extend T::Sig
  extend ActiveSupport::Concern

  TYPES = T.let(%w[
    CommentedPost
    CommentedRepost
    Repost
  ].freeze, T::Array[String])
end
