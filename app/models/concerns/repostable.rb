# typed: strict
# frozen_string_literal: true

module Repostable
  extend T::Sig
  extend ActiveSupport::Concern

  TYPES = T.let(%w[
    CommentedPost
    CommentedRepost
  ].freeze, T::Array[String])
end
