# typed: strict
# frozen_string_literal: true

class Internal::ApplicationForm
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes
end
