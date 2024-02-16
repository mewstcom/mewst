# typed: strict
# frozen_string_literal: true

module ControllerConcerns::Api::V1::FormErrorable
  # extend T::Sig
  # extend ActiveSupport::Concern
  #
  # sig { params(resource_class: T.class_of(V1::FormErrorResource), errors: ActiveModel::Errors).void }
  # def response_form_errors(resource_class:, errors:)
  #   resource_errors = resource_class.from_errors(errors: errors)
  #
  #   render(
  #     json: V1::FormErrorSerializer.new(resource_errors),
  #     status: :unprocessable_entity
  #   )
  # end
end
