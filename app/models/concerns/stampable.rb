# typed: strict
# frozen_string_literal: true

module Stampable
  extend T::Sig
  extend ActiveSupport::Concern

  TYPES = T.let(%w[
    CommentedPost
    CommentedRepost
  ].freeze, T::Array[String])
end
