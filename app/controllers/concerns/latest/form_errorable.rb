# typed: strict
# frozen_string_literal: true

module Latest::FormErrorable
  extend T::Sig
  extend ActiveSupport::Concern

  sig { params(resource_class: T.class_of(Latest::FormErrorResource), errors: ActiveModel::Errors).void }
  def response_form_errors(resource_class:, errors:)
    resource_errors = resource_class.from_errors(errors: errors)

    render(
      json: Latest::FormErrorSerializer.new(resource_errors),
      status: :unprocessable_entity
    )
  end
end
