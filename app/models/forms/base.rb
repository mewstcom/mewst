# typed: strict
# frozen_string_literal: true

class Forms::Base
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes
end
