# typed: strict
# frozen_string_literal: true

module Repostable
  extend T::Sig
  extend ActiveSupport::Concern

  TYPES = %w[
    CommentedPost
  ].freeze
end
