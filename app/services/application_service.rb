# typed: strict
# frozen_string_literal: true

class ApplicationService
  extend T::Sig

  include ActiveModel::Model
  include ActiveModel::Attributes

  def call
    raise NotImplementedError
  end
end
