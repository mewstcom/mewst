# typed: strict
# frozen_string_literal: true

module Profileable
  extend T::Sig
  extend ActiveSupport::Concern

  TYPES = T.let(%w[
    User
  ].freeze, T::Array[String])
end
