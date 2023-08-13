# typed: strict
# frozen_string_literal: true

class Internal::ApplicationForm
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes

  # @overload
  def self.i18n_scope
    :forms
  end
end
