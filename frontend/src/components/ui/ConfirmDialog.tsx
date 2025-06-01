import React from 'react';
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog'; // Assuming alert-dialog is already part of shadcn/ui setup
import { ButtonProps } from '@/components/ui/button'; // For confirmButtonVariant

interface ConfirmDialogProps {
    isOpen: boolean;
    onOpenChange: (open: boolean) => void;
    onConfirm: () => void;
    onCancel?: () => void;
    title: React.ReactNode;
    description: React.ReactNode;
    confirmText?: string;
    cancelText?: string;
    confirmButtonVariant?: ButtonProps['variant'];
}

export const ConfirmDialog: React.FC<ConfirmDialogProps> = ({
    isOpen,
    onOpenChange,
    onConfirm,
    onCancel,
    title,
    description,
    confirmText = 'Confirm',
    cancelText = 'Cancel',
}) => {
    if (!isOpen) {
        return null;
    }

    const handleConfirm = () => {
        onConfirm();
        onOpenChange(false); // Close dialog after confirm
    };

    const handleCancel = () => {
        if (onCancel) {
            onCancel();
        }
        onOpenChange(false); // Close dialog after cancel
    };

    return (
        <AlertDialog open={isOpen} onOpenChange={onOpenChange}>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{title}</AlertDialogTitle>
                    <AlertDialogDescription>{description}</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel onClick={handleCancel}>{cancelText}</AlertDialogCancel>
                    <AlertDialogAction onClick={handleConfirm}>
                        {confirmText}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}; 